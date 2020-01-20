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

	json, err := json.Marshal(locustConfig)
	if err != nil {
		message := fmt.Sprintf("error gettig locust config result: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// update test to 'Ready' status
	db.UpdateTestStatus(locustConfig.TestId, "Ready")

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func GetLocustConfigByTestId(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testId := ps.ByName("id")

	locustConfig, err := db.GetLocustConfigByTestId(testId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNoContent)
		return
	}

	json, err := json.Marshal(locustConfig)
	if err != nil {
		message := fmt.Sprintf("error gettig locust config result: " + err.Error())
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("locust config updated"))
}
