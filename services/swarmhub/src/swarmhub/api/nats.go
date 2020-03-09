package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"

	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/spf13/viper"
)

var sc stan.Conn
var natsUsername string
var natsPassword string
var natsURL string
var subStatus stan.Subscription
var stanClusterID string

type natsMessage struct {
	ID             string
	Params         map[string]string
	DeploymentType string
}

type deploymentStatus struct {
	ID             string
	DeploymentType string
	Status         string
	Params         map[string]string
}

func loadNatsSettings(conf *viper.Viper) {
	natsUsername = conf.GetString("NATS_USERNAME")
	natsPassword = conf.GetString("NATS_PASSWORD")
	natsURL = conf.GetString("NATS_URL")
	stanClusterID = conf.GetString("STAN_CLUSTER_ID")
}

func StartNats(config *viper.Viper) {
	loadNatsSettings(config)
	var natsServerURL string
	if natsPassword == "" || natsUsername == "" {
		fmt.Println("Nats without username/password.")
		natsServerURL = "nats://" + natsURL
	} else {
		natsServerURL = "nats://" + natsUsername + ":" + natsPassword + "@" + natsURL
	}

	if stanClusterID == "" {
		fmt.Println("Default stan cluster ID.")
		stanClusterID = "stan"
	}

	nc, err := nats.Connect(natsServerURL)
	if err != nil {
		log.Fatal("Failed to connect to nats: ", err)
	}

	hostname, err := os.Hostname()
	sc, err = stan.Connect(stanClusterID, hostname, stan.NatsConn(nc))
	if err != nil {
		log.Fatal("Failed to connect to nats streaming: ", err)
	}
	startTime := time.Now()

	// Used to receive messages on the state of deployments. Used to update the database
	// on the current state of a deployment.
	subStatus, err = sc.Subscribe("deployer.status", func(m *stan.Msg) {
		deployerStatusHandler(m)
	}, stan.StartAtTime(startTime))

	if err != nil {
		fmt.Println("Failed to subscribe to topic deployer.status: ", err.Error())
		os.Exit(2)
	}
}

func SendStartCmd(message []byte) error {
	err := sendStartCmd(message)
	return err
}

func sendStartCmd(message []byte) error {
	topic := "deployer.start"
	err := publishNatsMessage(sc, topic, message)
	if err != nil {
		return err
	}
	return nil
}

func SendStopCmd(message []byte) error {
	err := sendStopCmd(message)
	return err
}

func sendStopCmd(message []byte) error {
	topic := "deployer.stop"
	err := publishNatsMessage(sc, topic, message)
	if err != nil {
		return err
	}
	return nil
}

func publishNatsMessage(sc stan.Conn, topic string, message []byte) error {

	fmt.Println("publishing:", string(message), "To Topic:", topic)
	err := sc.Publish(topic, message)
	if err != nil {
		fmt.Println("Failed to publish message: ", err.Error())
		return err
	}
	fmt.Println("Message has been delivered.")
	return nil
}

func deployerStatusHandler(m *stan.Msg) {
	// createGrafanaSnapshot is run before updateDeployerStatus so grid information
	// is still attached to the test
	createGrafanaSnapshot(m)
	updateDeployerStatus(m)

}

func updateDeployerStatus(m *stan.Msg) {
	fmt.Printf("Msg received on [%s] : %s\n", m.Subject, string(m.Data))

	var deployedStatus deploymentStatus

	err := json.Unmarshal(m.Data, &deployedStatus)
	if err != nil {
		fmt.Println("failed to unmarshal struct")
		return
	}

	if deployedStatus.DeploymentType == "Test" {
		db.UpdateTestStatus(deployedStatus.ID, deployedStatus.Status)
		if deployedStatus.Status == "Error" {
			db.UpdateGridStatus(deployedStatus.Params["GRID_ID"], "Available")
		}
	} else if deployedStatus.DeploymentType == "Grid" || deployedStatus.DeploymentType == "DeleteGrid" {
		db.UpdateGridStatus(deployedStatus.ID, deployedStatus.Status)
		updateTestStatusFromGridStatus(deployedStatus.ID, deployedStatus.Status)
	} else if deployedStatus.DeploymentType == "StopTest" {
		// StopTest the ID is the gridID and the Param first value is the testID.
		if _, ok := deployedStatus.Params[KeyTestID]; !ok || deployedStatus.Params[KeyTestID] == "" {
			fmt.Println("Deployment type", deployedStatus.DeploymentType, "is expecting a parameter for testID.")
			return
		}
		gridID := deployedStatus.ID
		testID := deployedStatus.Params[KeyTestID]
		db.UpdateGridStatus(gridID, deployedStatus.Status)
		db.UpdateTestStatus(testID, deployedStatus.Status)
	} else {
		fmt.Println("Deployment status type of", deployedStatus.DeploymentType, "Was not expected.")
	}

}

func createGrafanaSnapshot(m *stan.Msg) {
	if !GrafanaEnabled {
		return
	}
	var deployedStatus deploymentStatus

	err := json.Unmarshal(m.Data, &deployedStatus)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal struct for deployedStatus: %v", err)
		fmt.Println(err)
		return
	}

	var testID string
	var status string

	switch deployedStatus.DeploymentType {
	case "Test":
		testID = deployedStatus.ID
		status = deployedStatus.Status
	case "StopTest":
		testID = deployedStatus.Params[KeyTestID]
		status = "Stopped"
	case "Grid":
		if !(deployedStatus.Status == "Deleted" || deployedStatus.Status == "expired") {
			return
		}
		status = "Stopped"
		testID, err = db.GetTestByGridID(deployedStatus.ID)
		if err != nil {
			err = fmt.Errorf("failed to GetTestByGridID using ID %v: %v", deployedStatus.ID, err)
			fmt.Println(err)
			return
		}

		if testID == "" {
			return
		}
	}

	if !(status == "Stopped" || status == "Expired") {
		// we don't want to create a snapshot if the status isn't stopped or expired
		return
	}

	name, startTime, endTime, err := db.InfoForGrafana(testID)
	if err != nil {
		err = fmt.Errorf("failed to get InfoForGrafana: %v", err)
		fmt.Println(err)
		return
	}

	go func() {
		// TODO: confirm if gridID is needed for generating snapshots
		err = generateGrafanaSnapshot(name, testID, "", startTime, endTime)
		if err != nil {
			err = fmt.Errorf("failed to generateGrafanaSnapshot: %v", err)
			fmt.Println(err)
			return
		}
	}()
}

// TODO:
func updateTestStatusFromGridStatus(gridID string, gridStatus string) {
	var testStatus string
	switch gridStatus {
	case "Expired":
		testStatus = "Expired"
	case "Deleted":
		testStatus = "Stopped"
	default:
		return
	}

	fmt.Printf("(grid: %v, status: %v) Update associated test to status %v\n", gridID, gridStatus, testStatus)
	testID, err := db.UpdateTestStatusThatUsesGrid(gridID, testStatus)
	if err != nil {
		err = fmt.Errorf("failed: %v", err)
		fmt.Println(err)
	}

	if testID != "" {
		fmt.Printf("test %v has been updated to status %v\n", testID, testStatus)
	}
}
