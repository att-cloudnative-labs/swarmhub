package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

var (
	tfstatebucket  string
	tfstateDirPath string
	scriptDirPath  string
	natsUsername   string
	natsPassword   string
	natsURL        string
	stanClusterID  string
	gridStatus     GridStatus
	sc             stan.Conn
	sleep          time.Duration
)

const (
	scriptName = "setup.sh"
)

type natsMessage struct {
	ID             string
	DeploymentType string
	Region         string
	Status         string
}

type GridStatus struct {
	grids map[string]string
	mux   sync.Mutex
}

func (c *GridStatus) Available(gridID string) (bool, string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	if val, ok := c.grids[gridID]; ok {
		return val == "Available", val
	}
	c.grids[gridID] = "Available"
	return true, "Available"
}

func (c *GridStatus) Update(gridID, status string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	switch status {
	case "Deleted", "Expired":
		if _, ok := c.grids[gridID]; ok {
			delete(c.grids, gridID)
		}
	default:
		c.grids[gridID] = status
	}
	return
}

func main() {
	gridStatus = GridStatus{grids: make(map[string]string)}
	setConfig()
	createNatsConnection()
	go subscribeForDeletions()

	go func() {
		for {
			if err := deleteExpiredInstances(); err != nil {
				log.Println(errors.WithMessage(err, "deleteExpiredInstances"))
			}
			time.Sleep(60 * 5 * time.Second)
		}

	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Prometheus metrics started.")
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
	sleep = registry.GetDuration("SLEEP")
	if sleep < 5*time.Second {
		fmt.Println("SLEEP is set to less than 5 seconds, using default 5 minutes instead.")
		sleep = time.Duration(5 * time.Minute)
	}
	tfstatebucket = registry.GetString("AWS_S3_BUCKET_TFSTATE")
	if tfstatebucket == "" {
		panic("AWS_S3_BUCKET_TFSTATE not defined")
	}
	tfstateDirPath = registry.GetString("TFSTATE_DIR_PATH")
	if tfstateDirPath == "" {
		panic("TFSTATE_DIR_PATH not defined")
	}
	scriptDirPath = registry.GetString("SCRIPT_DIR_PATH")
	if scriptDirPath == "" {
		panic("SCRIPT_DIR_PATH not definedL")
	}
}

func terraformOutput(statePath, key string) (string, error) {
	out, err := exec.Command("terraform", "output", "-state="+statePath, key).CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "exec.Command: %s", out)
	}
	if len(out) < 1 {
		return "", errors.New("empty output")
	}
	return strings.TrimSpace(string(out)), nil
}

func deleteExpiredInstances() error {
	log.Println("Syncing tfstate...")
	cmd := exec.Command("aws", "s3", "sync", "s3://"+tfstatebucket, tfstateDirPath, "--exclude", "*", "--include", "*-PROVISION", "--delete")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Println(errors.Wrapf(err, "cmd.CombinedOutput: %s", out))
		return nil
	}

	err := filepath.Walk(tfstateDirPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		go func() {
			ttl, err := terraformOutput(path, "ttl")
			if err != nil {
				log.Println(errors.WithMessage(err, "terraformOutput(ttl)"))
				return
			}
			secs, err := strconv.ParseInt(ttl, 10, 64)
			if err != nil {
				log.Println(errors.Wrap(err, "strconv.ParseInt"))
				return
			}
			gridID, err := terraformOutput(path, "grid_id")
			if err != nil {
				log.Println(errors.WithMessage(err, "terraformOutput(grid_id)"))
				return
			}
			if time.Unix(secs, 0).Before(time.Now()) {
				if avail, status := gridStatus.Available(gridID); !avail {
					log.Printf("Grid busy, postponed deletion: %v : %v", gridID, status)
					return
				}
				provider, err := terraformOutput(path, "provider")
				if err != nil {
					log.Println(errors.WithMessage(err, "terraformOutput(provider)"))
					return
				}
				if provider == "aws" {
				}
				cmd := exec.Command(scriptDirPath + "/" + scriptName)
				gridID, err := terraformOutput(path, "grid_id")
				if err != nil {
					log.Println(errors.WithMessage(err, "terraformOutput(grid_id)"))
					return
				}
				gridRegion, err := terraformOutput(path, "grid_region")
				if err != nil {
					log.Println(errors.WithMessage(err, "terraformOutput(gridRegion)"))
					return
				}
				log.Printf("Deleteing grid: %v", gridID)
				publishNatsMessage(gridRegion, gridID, "Deleting")
				cmd.Env = append(os.Environ(), []string{"GRID_ID=" + gridID, "GRID_REGION=" + gridRegion, "DESTROY=true", "PROVIDER=" + provider}...)
				output, err := cmd.CombinedOutput()
				if err != nil {
					publishNatsMessage(gridRegion, gridID, "Error")
					log.Println(errors.Wrapf(err, "cmd.CombinedOutput: %s", output))
					return
				}
				log.Printf("%s", output)
				publishNatsMessage(gridRegion, gridID, "Expired")
				log.Printf("Deleted grid: %v", gridID)
			}
		}()

		return nil
	})
	if err != nil {
		return errors.WithMessage(err, "filepath.Walk")
	}
	return nil
}

func publishNatsMessage(region string, gridID string, status string) {
	message := fmt.Sprintf(`{"ID": "%v", "Region": "%v", "DeploymentType": "Grid", "Status": "%v"}`, gridID, region, status)
	sc.Publish("deployer.status", []byte(message))
}

func createNatsConnection() {
	hostname, _ := os.Hostname()
	var err error
	sc, err = stan.Connect(stanClusterID, hostname, stan.NatsURL(natsURL))
	if err != nil {
		log.Println("Failed to connect to nats streaming cluster: ", err.Error())
		// Sleeping 60 seconds in order for the ec2 delete expired instances to be able to run.
		go func() {
			time.Sleep(60 * time.Second)
			os.Exit(2)
		}()
	}
}

func gridStatustracker(msg *stan.Msg) {
	var natsMsg natsMessage
	log.Printf("gridStatustracker message: %v", string(msg.Data))
	err := json.Unmarshal(msg.Data, &natsMsg)
	if err != nil {
		log.Println("Failed to unmarshal msg.Data", err.Error())
		return
	}

	if natsMsg.DeploymentType == "Grid" {
		gridStatus.Update(natsMsg.ID, natsMsg.Status)
	}
}

func subscribeForDeletions() {
	for {
		if _, err := sc.Subscribe("deployer.status", gridStatustracker); err != nil {
			log.Println("Failed to Subscribe to nats topic", err.Error())
		} else {
			break
		}
		time.Sleep(5 * time.Second)
	}
}
