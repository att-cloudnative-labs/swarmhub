package ec2

import (
	"encoding/json"
	"fmt"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/julienschmidt/httprouter"
)

func GetMasterIP(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Println("Getting master IP")
	testID := ps.ByName("id")
	type output struct {
		Status      string
		IP          string
		Auth        string
		Description string
	}

	cookie, err := r.Cookie("Authorization")
	if err != nil {
		_output := output{Status: "Failed", IP: "", Auth: "", Description: "Failed to get auth token from cookie."}
		b, _ := json.Marshal(_output)
		w.Write(b)
		return
	}

	cred := credentials.NewCredentials(&credentials.StaticProvider{Value: credentials.Value{
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}})

	// step 1. use testID and get grid ID and grid region
	gridID, gridRegion, err := db.GetGridByTestID(testID)
	if err != nil {
		_output := output{Status: "Failed", IP: "", Auth: "", Description: err.Error()}
		b, _ := json.Marshal(_output)
		w.Write(b)
		return
	}

	svc := ec2.New(session.New(&aws.Config{
		Region:      aws.String(gridRegion),
		Credentials: cred,
	}))

	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Grid"),
				Values: []*string{
					aws.String(gridID),
				},
			},
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String("locust-master"),
				},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
		},
	}

	result, err := svc.DescribeInstances(input)
	if err != nil {
		var errorMessage string
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				errorMessage = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			errorMessage = err.Error()
		}
		_output := output{Status: "Failed", IP: "", Auth: "", Description: errorMessage}
		b, _ := json.Marshal(_output)
		w.Write(b)
		return
	}

	if len(result.Reservations) > 0 {
		if len(result.Reservations[0].Instances) > 0 {
			ipAddress := *result.Reservations[0].Instances[0].PublicIpAddress
			_output := output{Status: "Success", IP: ipAddress, Auth: cookie.Value, Description: "Call was a success."}
			b, _ := json.Marshal(_output)
			w.Write(b)
			return
		}
	}

	_output := output{Status: "Failed", IP: "", Auth: "", Description: "No matching IP addresses."}
	b, _ := json.Marshal(_output)
	w.Write(b)
}
