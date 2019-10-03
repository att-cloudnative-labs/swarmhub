package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/nats-io/stan.go"
)

var PaginationItems = 10
var ShuttingDown bool

type DeploymentLog struct {
	ID         string
	StreamType string
	Output     string
	Running    bool
	Timestamp  int64
	Sequence   uint64
}

func deployerLogs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	subscriptionTopic := "deployer.output." + ps.ByName("id")

	if ShuttingDown {
		http.Error(w, "Can't view, in the process of shutting down.", http.StatusInternalServerError)
		return
	}

	fmt.Println("subscribing to topic: ", subscriptionTopic)

	var logsList []DeploymentLog
	var logTime int64

	sub, err := sc.Subscribe(subscriptionTopic, func(msg *stan.Msg) {
		deploymentLog := DeploymentLog{}
		err := json.Unmarshal(msg.Data, &deploymentLog)
		if err != nil {
			fmt.Println("Unable to convert deployment log to struct: ", err.Error())
		}
		logTime = msg.Timestamp
		deploymentLog.Timestamp = logTime / 1000000
		deploymentLog.Sequence = msg.Sequence

		logsList = append(logsList, deploymentLog)
	}, stan.DeliverAllAvailable())

	if err != nil {
		w.Write([]byte("Error subscribing to nats."))
		return
	}

	previouslyDelivered := int64(0)

	var currentTime int64
	for {
		time.Sleep(2 * time.Millisecond)
		currentTime = time.Now().UnixNano()
		delivered, err := sub.Delivered()
		pending, _, err := sub.Pending()
		if err != nil {
			fmt.Println("Error reading pending or delivered: ", err.Error())
			break
		}

		if delivered == 0 && pending == 0 {
			w.Write([]byte("No log messages."))
			sub.Unsubscribe()
			return
		}

		if (pending == 0 && delivered == previouslyDelivered) || logTime > currentTime {
			fmt.Printf("Delivered: %v, Previously Deliverred: %v, Current Time: %v, Log Time: %v\n", delivered, previouslyDelivered, currentTime, logTime)
			break
		}
		previouslyDelivered = delivered
	}
	sub.Unsubscribe()

	jsonResponse, err := json.Marshal(logsList)
	if err != nil {
		fmt.Println("Error converting logs to json: ", err.Error())
		return
	}

	w.Write(jsonResponse)
}

func Shutdown(c <-chan os.Signal) {
	<-c
	ShuttingDown = true
	fmt.Println("Shutting Down.")
	time.Sleep(2 * time.Second)
	subStatus.Close()
	sc.Close()
	os.Exit(0)
}
