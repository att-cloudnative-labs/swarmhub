package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	var grid db.GridStruct
	gridBytes, err := db.GetGridByID(id)
	if err != nil {
		fmt.Println("Unable to get test by ID: ", err.Error())
		return false, err
	}

	err = json.Unmarshal(gridBytes, &grid)
	if err != nil {
		fmt.Println("Error unmarshalling grid: ", err.Error())
		return false, err
	}

	if grid.Status == "Available" {
		return true, nil
	}

	return false, nil
}

func StartGrid(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var grid db.GridStruct
	id := ps.ByName("id")

	gridBytes, err := db.GetGridByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(gridBytes, &grid)
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

	ttlEpoch := strconv.FormatInt(time.Now().Add(time.Minute*time.Duration(ttl)).Unix(), 10)
	message := &natsMessage{ID: grid.ID, Cmd: "/ansible/gridProvision.sh", Params: []string{grid.ID, grid.Region, grid.Master, grid.Slave, grid.Nodes, ttlEpoch, LocustMasterSecurityGroups, LocustSlaveSecurityGroups}, DeploymentType: "Grid"}
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
	grid, err := db.GetGridByID(ps.ByName("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(grid)
}

func DeleteGrid(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	status, err := db.GetGridStatus(ps.ByName("id"))
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
		err := stopGrid(ps.ByName("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if status == "Deployed" || status == "Deploying" || status == "Available" {
		err := deleteDeployedGrid(ps.ByName("id"))
		// TODO: "This is a test to see if it prints."
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		err = db.DeleteGridByID(ps.ByName("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// w.Header().Set("Content-Type", "application/json")
}

func deleteDeployedGrid(id string) error {
	region, err := db.GetGridRegion(id)
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

	status := message{ID: id, DeploymentType: "Grid", Region: region, Status: "Deleting"}
	statusMsg, err := json.Marshal(status)
	if err != nil {
		fmt.Println("Failed to convert json: ", err.Error())
	}
	err = sc.Publish("deployer.status", statusMsg)
	if err != nil {
		fmt.Println("Was unable to update test status! ", err.Error())
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
