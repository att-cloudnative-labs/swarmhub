package api

import (
	"encoding/json"
	"fmt"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func CreateLocustConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var locustConfig db.LocustConfig

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&locustConfig)
	if err != nil {
		message := fmt.Sprintf("invalid locust config data")
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	locustConfig, err = db.CreateLocustConfig(locustConfig)
	if err != nil {
		message := fmt.Sprintf("error creating locust config: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(locustConfig)
	if err != nil {
		message := fmt.Sprintf("error gettig locust config result: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// update test to 'Ready' status
	db.UpdateTestStatus(locustConfig.TestId, "Ready")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonData)
}

func GetLocustConfigByTestId(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testId := ps.ByName("id")

	locustConfig, err := db.GetLocustConfigByTestId(testId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonData, err := json.Marshal(locustConfig)
	if err != nil {
		message := fmt.Sprintf("error gettig locust config result: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func UpdateLocustConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	var locustConfig db.LocustConfig

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&locustConfig)
	if err != nil {
		message := fmt.Sprintf("invalid locust config data")
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	err = db.UpdateLocustConfig(id, locustConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	testId := locustConfig.TestId
	if isTestRunning(testId) {
		// update running test
		gridID, gridRegion, err := db.GetGridByTestID(testId)
		if err != nil {
			message := fmt.Sprintf("error updating a running test: " + err.Error())
			http.Error(w, message, http.StatusInternalServerError)
			return
		}

		scriptID, scriptFileName, err := db.GetScriptFilename(testId)
		if err != nil {
			message := fmt.Sprintf("error updating a running test: " + err.Error())
			http.Error(w, message, http.StatusInternalServerError)
			return
		}

		params := map[string]string{
			"GRID_ID":          gridID,
			"GRID_REGION":      gridRegion,
			"GRID_AUTOSTART":   "true",
			"SCRIPT_ID":        scriptID,
			"SCRIPT_FILE_NAME": scriptFileName,
			"LOCUST_COUNT":     fmt.Sprint(locustConfig.Clients),
			"HATCH_RATE":       fmt.Sprint(locustConfig.HatchRate),
			"DEPLOYMENT":       "true",
		}
		message := &natsMessage{ID: testId, Params: params, DeploymentType: "Test"}
		b, err := json.Marshal(message)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("Not publishing nats message. Failed to convert to json: %v", err.Error())))
			return
		}

		err = sendStartCmd(b)
		if err != nil {
			message := fmt.Sprintf("error updating a running test: " + err.Error())
			http.Error(w, message, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("locust config updated"))
}

func isTestRunning(testId string) bool {
	var test db.Test
	testBytes, err := db.TestByID(testId)
	if err != nil {
		fmt.Println("Unable to get test by ID: ", err.Error())
		return false
	}

	err = json.Unmarshal(testBytes, &test)
	if err != nil {
		fmt.Println("Error unmarshalling test: ", err.Error())
		return false
	}

	if test.Status != "Deployed" {
		return false
	}

	return true
}
