package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/nats-io/stan.go"
)

var (
	cmdMap          = make(map[string]*CommandStruct)
	natsUsername    string
	natsPassword    string
	natsURL         string
	clusterID       string
	message         = make(chan string)
	sc              stan.Conn
	subCmdStart     stan.Subscription
	subDeployerStop stan.Subscription
	scriptPath      string
)

const (
	scriptName = "setup.sh"
	gridID     = "GRID_ID"
	testID     = "TEST_ID"
)

type CommandStruct struct {
	ID               string
	DeploymentType   string
	OutputStream     io.ReadCloser
	ErrorStream      io.ReadCloser
	cmd              *exec.Cmd
	Params           map[string]interface{}
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
	Params         map[string]interface{}
}

type DeploymentStatus struct {
	ID             string
	DeploymentType string
	Status         string
	Params         map[string]interface{}
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
	scriptPath = os.Getenv("SCRIPT_DIR_PATH")
	if scriptPath == "" {
		fmt.Println("Script path not set. Using /terraform")
		scriptPath = "/terraform"
	}
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		fmt.Printf("Script doesn not exist.")
		os.Exit(2)
	}
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
	stopCmdJob(stopMsg)
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
	fmt.Println("script path", scriptPath+"/"+scriptName)
	StartCmdJob(scriptPath+"/"+scriptName, startMsg.Params, startMsg.ID, startMsg.DeploymentType)

	fmt.Println("Finished running commands on start message ", startMsg.ID)
}

func startCmd(sc stan.Conn) {
	var err error
	for {
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

func StartCmdJob(command string, parameters map[string]interface{}, id string, deploymentType string) {
	runCommand(command, parameters, id, deploymentType)
}

func stopCmdJob(d Deployment) {
	var cid = cmdID(d.Params, d.ID, d.DeploymentType)
	if val, ok := cmdMap[cid]; ok {
		if err := val.cmd.Process.Signal(os.Interrupt); err != nil {
			fmt.Println("failed to stop process: ", err)
			return
		}
		output := CommandOutput{ID: d.ID, Output: "Stopping job...", Running: true, DeploymentType: cmdMap[cid].DeploymentType}
		pubMsg, err := json.Marshal(output)
		if err != nil {
			fmt.Println("Failed to convert stdout to json: ", err.Error())
		}
		fmt.Println("Publishing: ", string(pubMsg))
		publishTopic := "deployer.output." + d.ID
		sc.Publish(publishTopic, pubMsg)
		errUpdate := updateInitialDeploymentStatus(d.ID, d.DeploymentType, cmdMap[cid].Params)
		if errUpdate != nil {
			fmt.Println("Failed to update initial deployment status: ", err.Error())
		}
		return
	}

	fmt.Println("No Running process!")
	return
}

func runCommand(command string, parameters map[string]interface{}, id string, deploymentType string) error {
	var err error
	var cid = cmdID(parameters, id, deploymentType)
	if _, ok := cmdMap[cid]; ok {
		fmt.Println("Command already running.")
		return err
	}

	data := CommandStruct{}

	cmdMap[cid] = &data

	data.ID = cid
	data.cmd = exec.Command(command)
	data.Params = parameters
	data.DeploymentType = deploymentType
	// SysProcAttr being used to run commands as root
	//cmd.SysProcAttr = &syscall.SysProcAttr{}
	//cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0}
	data.cmd.Env = os.Environ()
	for key, val := range parameters {
		data.cmd.Env = append(data.cmd.Env, key+"="+fmt.Sprintf("%v", val))
	}
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
		if data.cmd.ProcessState.ExitCode() == 5 { //return exit code 5 from script if stopped by interupt signal
			var cancelType string
			switch cmdMap[cid].DeploymentType {
			case "Test":
				cancelType = "CancelTest"
			case "Grid":
				cancelType = "CancelGrid"
			}
			errUpdate = updateFinalDeploymentStatus(id, cancelType, parameters)
			if errUpdate != nil {
				errUpdate = fmt.Errorf("failed to updateDeploymentStatus: %v", errUpdate)
				fmt.Println(errUpdate)
			}
		} else {
			fmt.Println("Failed to run command: ", cmdErr.Error())
			status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Error", Params: parameters}
			statusMsg, err := json.Marshal(status)
			if err != nil {
				fmt.Println("Failed to convert json: ", err.Error())
			}
			sc.Publish("deployer.status", statusMsg)
		}
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

	delete(cmdMap, cid)
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

func updateInitialDeploymentStatus(id string, deploymentType string, parameters map[string]interface{}) error {
	var statusUpdates []DeploymentStatus
	switch deploymentType {
	case "Grid", "Test":
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Deploying", Params: parameters}
		statusUpdates = append(statusUpdates, status)
	case "DeleteGrid":
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Deleting", Params: parameters}
		statusUpdates = append(statusUpdates, status)
	case "CancelGrid":
		if val, ok := parameters[gridID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Grid", Status: "Stopping", Params: parameters})
		}
	case "StopTest":
		if val, ok := parameters[gridID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Grid", Status: "Cleaning", Params: parameters})
		}
		if val, ok := parameters[testID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Test", Status: "Stopping", Params: parameters})
		}
	case "CancelTest":
		if val, ok := parameters[gridID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Grid", Status: "Cleaning", Params: parameters})
		}
		if val, ok := parameters[testID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Test", Status: "Stopping", Params: parameters})
		}
	case "GridCleanup":
		status := DeploymentStatus{ID: id, DeploymentType: "Grid", Status: "Cleaning", Params: parameters}
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

func updateFinalDeploymentStatus(id string, deploymentType string, parameters map[string]interface{}) error {
	var statusUpdates []DeploymentStatus
	switch deploymentType {
	case "Grid":
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Available", Params: parameters}
		statusUpdates = append(statusUpdates, status)
	case "DeleteGrid":
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Deleted", Params: parameters}
		statusUpdates = append(statusUpdates, status)
	case "CancelGrid":
		if val, ok := parameters[gridID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Grid", Status: "Available", Params: parameters})
		}
	case "Test":
		status := DeploymentStatus{ID: id, DeploymentType: deploymentType, Status: "Deployed", Params: parameters}
		statusUpdates = append(statusUpdates, status)
	case "StopTest":
		if val, ok := parameters[gridID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Grid", Status: "Available", Params: parameters})
		}
		if val, ok := parameters[testID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Test", Status: "Stopped", Params: parameters})
		}
	case "CancelTest":
		if val, ok := parameters[gridID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Grid", Status: "Available", Params: parameters})
		}
		if val, ok := parameters[testID]; ok && val != "" {
			statusUpdates = append(statusUpdates, DeploymentStatus{ID: fmt.Sprintf("%v", val), DeploymentType: "Test", Status: "Stopped", Params: parameters})
		}
	case "GridCleanup":
		status := DeploymentStatus{ID: id, DeploymentType: "Grid", Status: "Available", Params: parameters}
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

func cmdID(parameters map[string]interface{}, id, deploymentType string) string {
	switch deploymentType {
	case "Test", "CancelTest":
		return fmt.Sprintf("%s-%s-%s", id, parameters["GRID_ID"], parameters["GRID_REGION"])
	case "Grid", "CancelGrid":
		return fmt.Sprintf("%s-%s", id, parameters["GRID_REGION"])
	}
	return ""
}
