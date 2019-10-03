package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

var (
	natsUsername  string
	natsPassword  string
	natsURL       string
	stanClusterID string
	sc            stan.Conn
	regions       []string
	sleep         time.Duration
)

type natsMessage struct {
	ID             string
	DeploymentType string
	Region         string
	Status         string
}

type ec2instance struct {
	ID   string
	Grid string
}

type ec2sessions struct {
	sessions map[string]ec2session
}

type ec2session struct {
	client *ec2.EC2
	region string
}

func main() {
	fmt.Println("ttl-enforcer")
	setConfig()
	createNatsConnection()

	sessions := createEC2Connections(regions)
	go func() {
		for {
			sessions.DeleteExpiredInstances()
			time.Sleep(60 * 5 * time.Second)
		}

	}()

	go sessions.SubscribeForDeletions()

	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Prometheus metrics started.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func setConfig() {
	registry := viper.New()
	registry.AutomaticEnv()
	registry.SetConfigName("settings")
	registry.AddConfigPath(".")
	if err := registry.MergeInConfig(); err != nil {
		err = fmt.Errorf("failed to MergeInConfig: %v", err)
		panic(err)
	}

	natsURLRaw := registry.GetString("NATS_URL")
	if natsURLRaw == "" {
		panic("Need to provide NATS_URL")
	}
	natsUsername = registry.GetString("NATS_USERNAME")
	natsPassword = registry.GetString("NATS_PASSWORD")
	if natsPassword == "" || natsUsername == "" {
		fmt.Println("Nats without username/password.")
		natsURL = "nats://" + natsURLRaw
	} else {
		natsURL = "nats://" + natsUsername + ":" + natsPassword + "@" + natsURLRaw
	}

	stanClusterID = registry.GetString("STAN_CLUSTER_ID")
	regions = registry.GetStringSlice("AWS_REGIONS")
	if len(regions) == 0 {
		panic("No regions to monitor!")
	}
	sleep = registry.GetDuration("SLEEP")
	if sleep < 5*time.Second {
		fmt.Println("SLEEP is set to less than 5 seconds, using default 5 minutes instead.")
		sleep = time.Duration(5 * time.Minute)
	}
}

func (s ec2sessions) terminationHandler(msg *stan.Msg) {
	var natsMsg natsMessage
	fmt.Println("terminationHanlder message:", string(msg.Data))

	err := json.Unmarshal(msg.Data, &natsMsg)
	if err != nil {
		fmt.Println("Failed to unmarshal msg.Data", err.Error())
		return
	}

	if natsMsg.DeploymentType == "Grid" && (natsMsg.Status == "Deleting") {
		go s.deleteGrid(natsMsg.Region, natsMsg.ID, 1)
	}
}

func (s ec2sessions) deleteGrid(region string, gridID string, tryCount int) {
	maxTries := 2
	session, ok := s.sessions[region]
	if !ok {
		fmt.Printf("Session does not exist for region %v. Region map is %v\n", region, s.sessions)
		return
	}
	instances, err := session.getGridInstances(gridID)
	if err != nil {
		fmt.Println("Failed to get grid instances from EC2", err.Error())
	}

	var instanceIDs []*string
	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			instanceIDs = append(instanceIDs, instance.InstanceId)
		}
	}

	// If no instance IDs are returned increment the retry counter, sleep, and then try again
	if len(instanceIDs) == 0 {
		if tryCount < maxTries {
			fmt.Printf("Did not find instances to delete for Region: %v, GridID: %v. Will try again in 60 seconds.\n", region, gridID)
			newTryCount := tryCount + 1
			time.Sleep(60 * time.Second)
			s.deleteGrid(region, gridID, newTryCount)
			return
		}
		fmt.Printf("Ending the search for instances to delete for Region: %v, GridID: %v\n", region, gridID)
		session.publishNatsMessage(region, gridID, "Deleted")
		return
	}

	instancesToTerminate := &ec2.TerminateInstancesInput{InstanceIds: instanceIDs}

	fmt.Printf("In Region %v for Grid %v deleting %v instances\n", region, gridID, len(instanceIDs))

	_, err = session.TerminateInstances(instancesToTerminate)
	if err != nil {
		fmt.Println("Failed to delete instances", err.Error())
	}

	// This means some instances were deleted, sleep and try one more time just in case.
	if tryCount < maxTries {
		fmt.Println("Will try again one last time in case of stragglers.")
		session.publishNatsMessage(region, gridID, "Deleted")
		newTryCount := tryCount + maxTries + 1
		time.Sleep(60 * time.Second)
		s.deleteGrid(region, gridID, newTryCount)
		return
	}

	fmt.Printf("Finished checking for instances to delete for Region: %v, Grid: %v\n", region, gridID)
}

// NatsDeletions is used to delete instances that were manually deleted
// in the UI of the tool instead of waiting for the TTL.
func (s ec2sessions) SubscribeForDeletions() {
	sub, err := sc.Subscribe("deployer.status", s.terminationHandler)
	if err != nil {
		fmt.Println("Failed to Subscribe to nats topic", err.Error())
	}
	_ = sub
}

func (s ec2sessions) getExpiredInstances() {
	for _, session := range s.sessions {
		session.getExpiredInstances()
	}
}

// DeleteExpiredInstances goes through the list of EC2 instances and terminates the instances
// that are expired based on TTL.
func (s ec2sessions) DeleteExpiredInstances() {
	for _, session := range s.sessions {
		session.deleteExpiredInstances()
	}
}

func (s ec2session) deleteExpiredInstances() error {
	instances, err := s.getExpiredInstances()
	if err != nil {
		fmt.Println("Failed to get expired instances")
		return err
	}
	if len(instances) == 0 {
		return nil
	}

	instanceIDs := []*string{}
	for _, val := range instances {
		instanceIDs = append(instanceIDs, &val.ID)
	}

	instancesToTerminate := ec2.TerminateInstancesInput{InstanceIds: instanceIDs}
	_, err = s.TerminateInstances(&instancesToTerminate)
	if err != nil {
		fmt.Println("Failed to delete instances.")
		return err
	}

	s.publishNatsMessagesFromEC2List(instances, "Expired")

	return nil
}

func (s ec2session) publishNatsMessage(region string, gridID string, status string) {
	message := fmt.Sprintf(`{"ID": "%v", "Region": "%v", "DeploymentType": "Grid", "Status": "%v"}`, gridID, region, status)
	sc.Publish("deployer.status", []byte(message))
}

func (s ec2session) publishNatsMessagesFromEC2List(instances []ec2instance, status string) {
	alreadyAdded := make(map[string]bool)
	for _, val := range instances {
		grid := val.Grid
		if alreadyAdded[grid] == false {
			s.publishNatsMessage(s.region, grid, status)
			alreadyAdded[grid] = true
		}
	}
}

func (s ec2session) getGridInstances(gridID string) (*ec2.DescribeInstancesOutput, error) {
	filters := []*ec2.Filter{
		&ec2.Filter{
			Name:   aws.String("tag:Grid"),
			Values: []*string{aws.String(gridID)},
		},
		&ec2.Filter{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("pending"), aws.String("running"), aws.String("rebooting"), aws.String("stopping"), aws.String("stopped")},
		},
	}

	request := &ec2.DescribeInstancesInput{Filters: filters}
	resp, err := s.DescribeInstances(request)
	return resp, err
}

func (s ec2session) getRunningTTLInstances() (*ec2.DescribeInstancesOutput, error) {
	filters := []*ec2.Filter{
		&ec2.Filter{
			Name:   aws.String("tag:TTL"),
			Values: []*string{aws.String("*")},
		},
		&ec2.Filter{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running")},
		},
	}

	request := &ec2.DescribeInstancesInput{Filters: filters}
	resp, err := s.DescribeInstances(request)
	return resp, err
}

func (s ec2session) getExpiredInstances() ([]ec2instance, error) {
	instances := []ec2instance{}
	resp, err := s.getRunningTTLInstances()
	if err != nil {
		fmt.Println("Failed to get list of instances from ec2")
		return instances, err
	}

	currentTime := time.Now().Unix()
	fmt.Printf("Current epoch time: %v\n", currentTime)
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			var ttl int64
			var gridID string
			for _, tag := range instance.Tags {
				key := *tag.Key
				value := *tag.Value
				var err error
				if key == "TTL" {
					ttl, err = strconv.ParseInt(value, 10, 64)
					if err != nil {
						fmt.Printf("Unable to convert TTL value of %v to int for instance %v. Setting TTL to 0.\n", value, *instance.InstanceId)
						ttl = 0
					}
				} else if key == "Grid" {
					gridID = value
				}
			}

			fmt.Printf("Instance ID: %v, Grid ID: %v, TTL: %v\n", *instance.InstanceId, gridID, ttl)

			if ttl < currentTime {
				fmt.Println("Expired, adding to list.")
				instances = append(instances, ec2instance{ID: *instance.InstanceId, Grid: gridID})
			}

		}
	}

	return instances, nil
}

func createNatsConnection() {
	hostname, _ := os.Hostname()
	var err error
	sc, err = stan.Connect(stanClusterID, hostname, stan.NatsURL(natsURL))
	if err != nil {
		fmt.Println("Failed to connect to nats streaming cluster: ", err.Error())
		// Sleeping 60 seconds in order for the ec2 delete expired instances to be able to run.
		go func() {
			time.Sleep(60 * time.Second)
			os.Exit(2)
		}()
	}
}

func createEC2Connections(regions []string) ec2sessions {
	client := make(map[string]ec2session)
	for _, region := range regions {
		svc := ec2.New(session.New(&aws.Config{
			Region: aws.String(region),
		}))
		client[region] = ec2session{client: svc, region: region}
	}

	return ec2sessions{client}
}

func (s ec2session) TerminateInstances(instancesToTerminate *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	output, err := s.client.TerminateInstances(instancesToTerminate)
	return output, err
}

func (s ec2session) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	output, err := s.client.DescribeInstances(input)
	return output, err
}

func (s ec2session) deleteInstances(instanceIDs []*string) error {
	if len(instanceIDs) == 0 {
		return nil
	}

	instancesToTerminate := ec2.TerminateInstancesInput{InstanceIds: instanceIDs}
	_, err := s.TerminateInstances(&instancesToTerminate)
	if err != nil {
		fmt.Println("Failed to delete instances.")
		fmt.Println(err)
		return err
	}

	return nil
}
