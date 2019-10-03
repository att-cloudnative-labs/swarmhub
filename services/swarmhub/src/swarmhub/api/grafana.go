package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/api/grafana/snapshot"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"

	"github.com/julienschmidt/httprouter"
)

var (
	GrafanaEnabled      bool
	GrafanaDomain       string
	GrafanaAPIKey       string
	GrafanaDashboardUID string
)

func GrafanaConfigs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if !GrafanaEnabled {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if GrafanaDashboardUID == "" || GrafanaDomain == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	payload := fmt.Sprintf("{\"Enabled\": %t, \"BaseURL\":\"%s\", \"DashboardUID\":\"%s\"}", GrafanaEnabled, GrafanaDomain, GrafanaDashboardUID)
	w.Write([]byte(payload))
}

// GenerateGrafanaSnapshotHandler is used to generate a snapshot
func GenerateGrafanaSnapshotHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if !GrafanaEnabled {
		return
	}
	decoder := json.NewDecoder(r.Body)

	type grafana struct {
		TestID string
	}

	var snapshot grafana

	err := decoder.Decode(&snapshot)
	if err != nil {
		err = fmt.Errorf("error decoding payload: %v", err)
		http.Error(w, "Need to provide Name, Start, and End fields", http.StatusInternalServerError)
		return
	}

	err = generateGrafanaSnapshotFromTestID(snapshot.TestID)
	if err != nil {
		err = fmt.Errorf("failed to generateGrafanaSnapshot: %v", err)
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

// generateGrafanaSnapshotWithInfo is created to make the call that has the information that is needed,
// so it can be run in a goroutine and it doesn't matter if the grid id is deleted.
func generateGrafanaSnapshot(name, testID, gridID string, startTime, endTime time.Time) error {
	grafanaDomain, err := url.Parse(GrafanaDomain)

	config := &snapshot.Config{
		GrafanaAddr:   grafanaDomain,
		GrafanaAPIKey: GrafanaAPIKey,
	}

	snapclient, err := snapshot.NewSnapClient(config)
	if err != nil {
		err = fmt.Errorf("failed to create NewSnapClient: %v", err)
		return err
	}

	if gridID == "" {
		gridID = ".*"
	}
	takeConfig := &snapshot.TakeConfig{
		DashUID:  GrafanaDashboardUID,
		DashSlug: name,
		From:     &startTime,
		To:       &endTime,
		Vars:     map[string]string{"gridID": gridID},
	}

	snapshot, err := snapclient.Take(takeConfig)
	if err != nil {
		err = fmt.Errorf("failed to snapclient.Take: %v", err)
		return err
	}

	err = db.GrafanaSnapshot(testID, snapshot.URL, snapshot.DeleteURL)
	if err != nil {
		err = fmt.Errorf("failed to create GrafanaSnapshot: %v", err)
		return err
	}

	return nil
}

func generateGrafanaSnapshotFromTestID(testID string) error {
	name, gridID, startTime, endTime, err := db.InfoForGrafana(testID)
	if err != nil {
		err = fmt.Errorf("failed to get InfoForGrafana: %v", err)
		return err
	}

	err = generateGrafanaSnapshot(name, testID, gridID, startTime, endTime)
	if err != nil {
		err = fmt.Errorf("failed to generateGrafanaSnapshot: %v", err)
		return err
	}

	return nil
}
