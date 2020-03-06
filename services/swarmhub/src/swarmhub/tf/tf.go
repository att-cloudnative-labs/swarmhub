package tf

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

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
		AccessKeyID:     os.Getenv("AWS_S3_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("AWS_S3_SECRET_ACCESS_KEY"),
	}})

	// step 1. use testID and get grid ID and grid region
	gridID, gridRegion, err := db.GetGridByTestID(testID)
	if err != nil {
		_output := output{Status: "Failed", IP: "", Auth: "", Description: err.Error()}
		b, _ := json.Marshal(_output)
		w.Write(b)
		return
	}

	svc := s3.New(session.New(&aws.Config{
		Region:      aws.String(os.Getenv("AWS_S3_REGION")),
		Credentials: cred,
	}))

	objName := fmt.Sprintf("%s-%s-PROVISION", gridID, gridRegion)

	input := &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("AWS_S3_BUCKET_TFSTATE")),
		Key:    aws.String(objName),
	}

	result, err := svc.GetObject(input)
	if err != nil {
		var errorMessage string
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			//case s3.ErrCodeNoSuchKey:
			//	fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
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

	stateFilePath := os.Getenv("TFSTATE_DIR_PATH") + "/" + objName
	statefile, err := os.Create(stateFilePath)
	if err != nil {
		_output := output{Status: "Failed", IP: "", Auth: "", Description: err.Error()}
		b, _ := json.Marshal(_output)
		w.Write(b)
		return

	}

	if _, err := io.Copy(statefile, result.Body); err != nil {
		_output := output{Status: "Failed", IP: "", Auth: "", Description: err.Error()}
		b, _ := json.Marshal(_output)
		w.Write(b)
		return
	}

	defer func() {
		if err := statefile.Close(); err != nil {
			fmt.Println(err)
		}
		// remove state file after
		if err := os.Remove(stateFilePath); err != nil {
			fmt.Println(err)
		}
		return
	}()

	locustMasterIP, err := terraformOutput(stateFilePath, "locust_master_ip")
	if err != nil {
		_output := output{Status: "Failed", IP: "", Auth: "", Description: errors.WithMessage(err, "terraformOutput(locust_master_ip)").Error()}
		b, _ := json.Marshal(_output)
		w.Write(b)
		return
	}
	_output := output{Status: "Success", IP: locustMasterIP, Auth: cookie.Value, Description: "Call was a success."}
	b, _ := json.Marshal(_output)
	w.Write(b)
	return
}
