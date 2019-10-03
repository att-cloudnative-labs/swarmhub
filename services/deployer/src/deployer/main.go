package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nats-io/stan.go"
)

var (
	cmdMap = make(map[string]*CommandStruct)
	natsUsername string
	natsPassword string
	natsURL string
	clusterID string
	startCmdMutex = &sync.Mutex{}
	message = make(chan string)
	sc stan.Conn
	subCmdStart stan.Subscription
	subDeployerStop stan.Subscription
)



type CommandStruct struct {
	ID               string
	DeploymentType   string
	OutputStream     io.ReadCloser
	ErrorStream      io.ReadCloser
	cmd              *exec.Cmd
	CurrentlyRunning bool
}

type CommandOutput struct {
	ID             string
	DeploymentType string
	StreamType     string
	Output         string
	Running        bool
}

type Deployment struct {
	ID             string
	DeploymentType string
	Cmd            string
	Params         []string
}

type DeploymentStatus struct {
	ID             string
	DeploymentType string
	Status         string
	Params         []string
}

func loadNatsFromEnv() {
	natsUsername = os.Getenv("NATS_USERNAME")
	natsPassword = os.Getenv("NATS_PASSWORD")
	natsURLRaw := os.Getenv("NATS_URL")
	if natsURLRaw == "" {
		fmt.Println("Using default nats url.")
		natsURLRaw = "nats.swarmhub.svc.cluster.local:4222"
	}

	if natsPassword == "" || natsUsername == "" {
		fmt.Println("Nats without username/password.")
		natsURL = "nats://" + natsURLRaw
	} else {
		natsURL = "nats://" + natsUsername + ":" + natsPassword + "@" + natsURLRaw
	}

	clusterID = os.Getenv("STAN_CLUSTER_ID")
	if clusterID == "" {
		fmt.Println("Default stan cluster ID.")
		clusterID = "stan"
	}
}

func init() {
	loadNatsFromEnv()
}

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Failed to get Hostname: %v\n", err)
		os.Exit(2)
	}
	sc, err = stan.Connect(clusterID, hostname, stan.NatsURL(natsURL))
	if err != nil {
		fmt.Printf("Failed to connect to nats streaming cluster: %v", err)
		os.Exit(2)
	}

	go startCmd(sc)
	subDeployerStop, _ = sc.Subscribe("deployer.stop", messageStopHandler)

	signal_chan := make(chan os.Signal)
	signal.Notify(signal_chan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
	shutdown(sc, signal_chan)
}

func shutdown(sc stan.Conn, c <-chan os.Signal) {
	<-c
	fmt.Println("Shutting Down.")
	subCmdStart.Close()
	subDeployerStop.Close()
	sc.Close()
	os.Exit(0)
}

func messageStopHandler(msg *stan.Msg) {
	fmt.Println("Recieved stop message: ", string(msg.Data))
	var stopMsg Deployment

	err := json.Unmarshal(msg.Data, &stopMsg)
	if err != nil {
		fmt.Println("failed to unmarshal msg.Data: ", err.Error())
		return
	}
	stopCmdJob(stopMsg.ID)
	fmt.Println("Finished running stop handler for ", stopMsg.ID)
}

// Possibly a very small gap that additional message might get sent before unsubscribed?
func messageStartHandler(msg *stan.Msg) {
	msg.Ack()
	message <- "started"
	fmt.Println("Recieved start message: ", string(msg.Data))
	var startMsg Deployment

	err := json.Unmarshal(msg.Data, &startMsg)
	fmt.Println("startMsg is: ", startMsg)
	if err != nil {
		fmt.Println("failed to unmarshal msg.Data: ", err.Error())
	}

	StartCmdJob(startMsg.Cmd, startMsg.Params, startMsg.ID, startMsg.DeploymentType)

	fmt.Println("Finished running commands on start message ", startMsg.ID)
	startCmdMutex.Unlock()
}

func startCmd(sc stan.Conn) {
	var err error
	for {
		startCmdMutex.Lock()
		fmt.Println("Subscribing to deployer.start")
		subCmdStart, err = sc.QueueSubscribe("deployer.start", "deployer", messageStartHandler,
			stan.DurableName("durable-deployer"), stan.SetManualAckMode(), stan.MaxInflight(1))
		if err != nil {
			fmt.Println("Error creating subscription: ", err.Error())
			os.Exit(1)
		}
		_ = <-message
		fmt.Println("Closing deployer.start")
		err = subCmdStart.Close()
		if err != nil {
			fmt.Println("Error closing connection to deployer.start: ", err.Error())
		}

	}
}

func StartCmdJob(command string, parameters []string, id string, deploymentType string) {
	runCommand(command, parameters, id, deploymentType)
}

func stopCmdJob(id string) {
	if val, ok := cmdMap[id]; ok {
		if err := val.cmd.Process.Kill(); err != nil {
			fmt.Println("failed to kill process: ", err)
			return
		}
		output := CommandOutput{ID: id, Output: "Stopped job.", Running: false, DeploymentType: cmdMap[id].DeploymentType}
		pubMsg, err := json.Marshal(output)
		if err != nil {
			fmt.Println("Failed to convert stdout to json: ", err.Error())
		}
		fmt.Println("Publishing: ", string(pubMsg))
		publishTopic := "deployer.output." + id
		sc.Publish(publishTopic, pubMsg)
		return
	}

	fmt.Println("No Running process!")
	return
}

func runCommand(command string, parameters []string, id string, deploymentType string) error {
	var err error

	if _, ok := cmdMap[id]; ok {
		fmt.Println("Command already running.")
		return err
	}

	data := CommandStruct{}

	cmdMap[id] = &data

	data.ID = id
	data.cmd = exec.Command(command, parameters...)
	// SysProcAttr being used to run commands as root
	//cmd.SysProcAttr = &syscall.SysProcAttr{}
	//cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0}
	data.cmd.Env = os.Environ()
	data.OutputStream, err = data.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	data.ErrorStream, err = data.cmd.StderrPipe()
	if err != nil {
		return err
	}
	publishTopic := "deployer.output." + id
	fmt.Println("Publish topic is: ", publishTopic)

	go CommandStdOutput(data, publishTopic)
	go CommandStdError(data, publishTopic)

	data.cmd.Start()
	data.CurrentlyRunning = true
	errUpdate := updateInitialDeploymentStatus(id, deploymentType, parameters)
	if errUpdate != nil {
		fmt.Println("Failed to update initial deployment status: ", err.Error())
	}

	cmdErr := data.cmd.Wait()
	if cmdErr != nil {
		fmt.Println("Failed to run command: ", cmdErr.Error())
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Error", Params: parameters}
		statusMsg, err := json.Marshal(status)
		if err != nil {
			fmt.Println("Failed to convert json: ", err.Error())
		}
		sc.Publish("deployer.status", statusMsg)
	}
	data.CurrentlyRunning = false
	output := CommandOutput{ID: data.ID, Running: data.CurrentlyRunning}
	pubMsg, err2 := json.Marshal(output)
	if err2 != nil {
		fmt.Println("Failed to convert stdout to json: ", err.Error())
	}
	fmt.Println("Publishing: ", string(pubMsg))
	sc.Publish(publishTopic, pubMsg)
	sc.Publish("deployer.done", pubMsg)

	if cmdErr == nil {
		errUpdate = updateFinalDeploymentStatus(id, deploymentType, parameters)
		if errUpdate != nil {
			errUpdate = fmt.Errorf("failed to updateDeploymentStatus: %v", errUpdate)
			fmt.Println(errUpdate)
		}
	}

	delete(cmdMap, id)
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0

			// This works on both Unix and Windows. Although package
			// syscall is generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in both cases has
			// an ExitStatus() method with the same signature.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				fmt.Printf("Exit Status: %d\n", status.ExitStatus())
			}
		} else {
			fmt.Printf("cmd.Wait: %v", err)
		}
	}
	return err
}

func updateInitialDeploymentStatus(id string, deploymentType string, parameters []string) error {
	var statusUpdates []DeploymentStatus
	switch deploymentType {
	case "Grid", "Test":
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Deploying"}
		statusUpdates = append(statusUpdates, status)
	case "StopTest":
		// if there is a grid associated to the deployment mark it as cleaning
		if len(parameters) > 0 && parameters[0] != "" {
			gridID := parameters[0]
			status := DeploymentStatus{ID: gridID, DeploymentType: "Grid", Status: "Cleaning"}
			statusUpdates = append(statusUpdates, status)
		}
		if len(parameters) > 2 && parameters[2] != "" {
			testID := parameters[2]
			status := DeploymentStatus{ID: testID, DeploymentType: "Test", Status: "Stopping"}
			statusUpdates = append(statusUpdates, status)
		}
	case "CancelTest":
		if len(parameters) > 0 && parameters[0] != "" {
			gridID := parameters[0]
			status := DeploymentStatus{ID: gridID, DeploymentType: "Grid", Status: "Cleaning"}
			statusUpdates = append(statusUpdates, status)
		}
	case "GridCleanup":
		status := DeploymentStatus{ID: id, DeploymentType: "Grid", Status: "Cleaning"}
		statusUpdates = append(statusUpdates, status)
	}

	for i, status := range statusUpdates {
		err := deploymentMessage(status)
		if err != nil {
			err = fmt.Errorf("deploymentMessage %v of %v failed: %v", i+1, len(statusUpdates), err)
			return err
		}
	}
	return nil
}

func updateFinalDeploymentStatus(id string, deploymentType string, parameters []string) error {
	var statusUpdates []DeploymentStatus
	switch deploymentType {
	case "Grid":
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Available"}
		statusUpdates = append(statusUpdates, status)
	case "Test":
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Deployed"}
		statusUpdates = append(statusUpdates, status)
	case "StopTest":
		// if there is a grid associated to the deployment mark it available
		if len(parameters) > 0 && parameters[0] != "" {
			gridID := parameters[0]
			status := DeploymentStatus{ID: gridID, DeploymentType: "Grid", Status: "Available"}
			statusUpdates = append(statusUpdates, status)
		}

		if len(parameters) > 2 && parameters[2] != "" {
			testID := parameters[2]
			status := DeploymentStatus{ID: testID, DeploymentType: "Test", Status: "Stopped"}
			statusUpdates = append(statusUpdates, status)
		}
	case "CancelTest":
		// if there is a grid associated to the deployment mark it available
		if len(parameters) > 0 && parameters[0] != "" {
			gridID := parameters[0]
			status := DeploymentStatus{ID: gridID, DeploymentType: "Grid", Status: "Available"}
			statusUpdates = append(statusUpdates, status)
		}
	case "GridCleanup":
		status := DeploymentStatus{ID: id, DeploymentType: "Grid", Status: "Available"}
		statusUpdates = append(statusUpdates, status)
	}

	for i, status := range statusUpdates {
		err := deploymentMessage(status)
		if err != nil {
			err = fmt.Errorf("deploymentMessage %v of %v failed: %v", i+1, len(statusUpdates), err)
			return err
		}
	}
	return nil
}

func deploymentMessage(status DeploymentStatus) error {
	statusMsg, err := json.Marshal(status)
	if err != nil {
		err = fmt.Errorf("failed to convert json: %v", err)
		return err
	}
	err = sc.Publish("deployer.status", statusMsg)
	if err != nil {
		err = fmt.Errorf("failed to publish message: %v", err)
		return err
	}
	return nil
}

func CommandStdOutput(cmd CommandStruct, publishTopic string) {
	output := CommandOutput{ID: cmd.ID, Running: true, StreamType: "stdin"}
	scanner := bufio.NewScanner(cmd.OutputStream)
	for scanner.Scan() {
		output.Output = scanner.Text()
		data, err := json.Marshal(output)
		if err != nil {
			fmt.Println("Failed to convert stdout to json: ", err.Error())
		}
		fmt.Println("Publishing: ", string(data))
		err = sc.Publish(publishTopic, data)
		if err != nil {
			fmt.Println("Failed to publish to topic: ", err.Error())
		}
	}
}

func CommandStdError(cmd CommandStruct, publishTopic string) {
	output := CommandOutput{ID: cmd.ID, Running: true, StreamType: "stderr"}
	scanner := bufio.NewScanner(cmd.ErrorStream)
	for scanner.Scan() {
		output.Output = scanner.Text()
		data, err := json.Marshal(output)
		if err != nil {
			fmt.Println("Failed to convert stderr to json: ", err.Error())
		}
		fmt.Println("Publishing: ", string(data))
		err = sc.Publish(publishTopic, data)
		if err != nil {
			fmt.Println("Failed to publish to topic: ", err.Error())
		}
	}
}
