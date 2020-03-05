package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/jwt"

	"github.com/julienschmidt/httprouter"
)

var (
	LocustMasterSecurityGroups string
	LocustSlaveSecurityGroups  string
)

func validateCanRunGrid(id string) (bool, error) {
	grid, err := db.GetGridById(id)
	if err != nil {
		fmt.Println("Unable to validate if grid can run: ", err.Error())
		return false, err
	}

	if grid.Status == "Available" {
		return true, nil
	}

	return false, nil
}

func StartGrid(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")

	grid, err := db.GetGridById(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if grid.Status != "Ready" {
		w.WriteHeader(http.StatusUnauthorized)
		http.Error(w, fmt.Sprintf("Grid has a status %v, needs to be 'Ready'", grid.Status), http.StatusUnauthorized)
		return
	}

	err = db.UpdateGridStatus(id, "Deploying")
	if err != nil {
		// Print error but continue on
		fmt.Printf("Failed to update grid status for %v, %v\n", id, err)
	}

	ttl, err := strconv.Atoi(grid.TTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slaveCore, err := db.GetInstanceCore(grid.Region, grid.Slave)
	if err != nil {
		// Print error but continue on
		fmt.Printf("Failed to get slave core %v\n", err)
	}

	ttlEpoch := strconv.FormatInt(time.Now().Add(time.Minute*time.Duration(ttl)).Unix(), 10)
	params := map[string]string{
		"GRID_ID":                      grid.ID,
		"GRID_REGION":                  grid.Region,
		"MASTER_INSTANCE":              grid.Master,
		"SLAVE_INSTANCE":               grid.Slave,
		"SLAVE_INSTANCE_CORE":          strconv.Itoa(slaveCore),
		"SLAVE_INSTANCE_COUNT":         grid.Nodes,
		"PROVIDER":                     strings.ToLower(grid.Provider),
		"TTL":                          ttlEpoch,
		"LOCUST_MASTER_SECURITY_GROUP": LocustMasterSecurityGroups,
		"LOCUST_SLAVE_SECURITY_GROUP":  LocustSlaveSecurityGroups,
		"PROVISION":                    "true",
	}
	message := &natsMessage{ID: grid.ID, Params: params, DeploymentType: "Grid"}
	b, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Not publishing nats message. Failed to convert to json: ", err.Error())
		return
	}

	err = sendStartCmd(b)
	if err != nil {
		fmt.Println("Was unable to update test status! ", err.Error())
		return
	}
	return
}

func stopGrid(id string) error {
	message := &natsMessage{ID: id, DeploymentType: "Grid"}
	b, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Not publishing nats message. Failed to convert to json: ", err.Error())
		return err
	}
	err = sendStopCmd(b)
	if err != nil {
		fmt.Println("Failed to send stop command for grid: " + id)
		return err
	}

	return nil
}

func deleteGrid(grid db.GridStruct) error {
	params := map[string]string{
		"GRID_REGION": grid.Region,
		"GRID_ID":     grid.ID,
		"PROVIDER":    strings.ToLower(grid.Provider),
		"DESTROY":     "true",
	}
	message := &natsMessage{ID: grid.ID, Params: params, DeploymentType: "DeleteGrid"}
	b, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Not publishing nats message. Failed to convert to json: ", err.Error())
		return err
	}
	err = sendStartCmd(b)
	if err != nil {
		fmt.Println("Failed to send delete command for grid: " + grid.ID)
		return err
	}

	return nil
}

// incomplete
func StopGrid(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := stopGrid(ps.ByName("id"))
	if err != nil {
		w.Write([]byte("Failed to perform StopGrid: " + ps.ByName("id")))
		return
	}
	w.Write([]byte("sent a stop command for grid id: " + ps.ByName("id")))
	return
}

func Grids(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var grids []byte
	var err error
	var itemsPerPage int

	itemsPerPageStr := r.URL.Query().Get("items")
	if itemsPerPageStr == "" {
		itemsPerPage = PaginationItems
	} else {
		itemsPerPage, err = strconv.Atoi(itemsPerPageStr)
		if err != nil {
			itemsPerPage = PaginationItems
		}
	}

	status, ok := r.URL.Query()["status"]
	if ok && len(status[0]) > 0 {
		grids, err = db.GetGridsByStatus(status[0])
	} else {
		grids, err = db.GetGrids(itemsPerPage)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(grids)
}

func Grid(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	grid, err := db.GetGridById(ps.ByName("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(grid)
	if err != nil {
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func DeleteGrid(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	status, err := db.GetGridStatus(id)
	if err != nil {
		err = fmt.Errorf("unable to extract the id from Deletegrid function: %v", err)
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// TODO:
	// add if statement here when we want to add logic for different
	// grid status before deleting. For simplicity just delete now.

	// add if statement to check if linked to a test and change status of both
	// the test and the grid

	if status == "Deploying" {
		message := fmt.Sprintf("Grid is being deployed")
		http.Error(w, message, http.StatusProcessing)
		return
		/* grid, err := db.GetGridById(id)
		if err != nil {
			message := fmt.Sprintf("Error deleting grid: " + err.Error())
			http.Error(w, message, http.StatusInternalServerError)
			return
		}

		err = stopGrid(grid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} */
	}

	if status == "Deployed" || status == "Available" || status == "Ready" || status == "Error" {
		err := deleteDeployedGrid(id)
		// TODO: "This is a test to see if it prints."
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = db.DeleteGridByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// w.Header().Set("Content-Type", "application/json")
}

func deleteDeployedGrid(id string) error {
	grid, err := db.GetGridById(id)
	if err != nil {
		fmt.Println("Unable to extract the id from Deletegrid function", err.Error())
		return err
	}

	type message struct {
		ID             string
		DeploymentType string
		Region         string
		Status         string
	}

	status := message{ID: id, DeploymentType: "Grid", Region: grid.Region, Status: "Deleting"}
	statusMsg, err := json.Marshal(status)
	if err != nil {
		fmt.Println("Failed to convert json: ", err.Error())
	}
	err = sc.Publish("deployer.status", statusMsg)
	if err != nil {
		fmt.Println("Was unable to update test status! ", err.Error())
		return err
	}

	err = deleteGrid(grid)
	if err != nil {
		return err
	}

	fmt.Println("Command sent to nats for deleting grid ID:", id)
	return nil
}

func PaginateGridInfo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type paginateInfo struct {
		FirstGrid     string
		LastGrid      string
		NumberOfGrids int
	}

	firstGrid, err := db.GetFirstGrid()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lastGrid, err := db.GetLastGrid()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	numberOfGrids, err := db.GetNumberOfGrids()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := paginateInfo{firstGrid, lastGrid, numberOfGrids}
	b, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func GridsPaginate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	gridID := ps.ByName("id")
	var itemsPerPage int
	var err error

	itemsPerPageStr := r.URL.Query().Get("items")
	if itemsPerPageStr == "" {
		itemsPerPage = PaginationItems
	} else {
		itemsPerPage, err = strconv.Atoi(itemsPerPageStr)
		if err != nil {
			itemsPerPage = PaginationItems
		}

	}

	grids, err := db.GridPaginate(gridID, itemsPerPage)
	if err != nil {
		fmt.Println("Error grabbing data.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(grids)
}

func GetGridPaginateKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	var offset int
	var err error

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offset = 0
	} else {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			offset = 0
		}
	}

	testID, err := db.GridGetPaginateKey(id, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(testID))
}

func GetGridProviderTypes(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	providers, err := db.GetGridProviders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(providers)

}

func GetGridInstanceTypes(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	provider, ok := r.URL.Query()["provider"]
	if !ok || len(provider[0]) < 1 {
		http.Error(w, "Url Param 'provider' is missing", http.StatusInternalServerError)
		return
	}

	region, ok := r.URL.Query()["region"]
	if !ok || len(region[0]) < 1 {
		http.Error(w, "Url Param 'region' is missing", http.StatusInternalServerError)
		return
	}
	grids, err := db.GetGridInstances(provider[0], region[0])
	if err != nil {
		http.Error(w, "Unable to get grid instances", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(grids)
}

func GetGridRegionTypes(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	provider, ok := r.URL.Query()["provider"]
	if !ok || len(provider[0]) < 1 {
		http.Error(w, "Url Param 'provider' is missing", http.StatusInternalServerError)
		return
	}

	regions, err := db.GetGridRegions(provider[0])
	if err != nil {
		http.Error(w, "Unable to get grid instances", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(regions)
}

func CreateGrid(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := jwt.TokenAudienceFromRequest(r)
	decoder := json.NewDecoder(r.Body)

	type createGridStruct struct {
		Name       string
		Provider   string
		Region     string
		MasterType string
		SlaveType  string
		SlaveNodes int
		TTL        int
	}

	var grid createGridStruct

	err := decoder.Decode(&grid)

	if err != nil {
		fmt.Println("Error decoding payload: ", err)
		return
	}

	db.CreateGrid(grid.Name, grid.Provider, grid.Region, grid.MasterType, grid.SlaveType, grid.SlaveNodes, grid.TTL, user)
}
