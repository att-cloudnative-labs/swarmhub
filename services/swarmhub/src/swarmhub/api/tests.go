package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/db"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/jwt"
	"github.com/att-cloudnative-labs/swarmhub/services/swarmhub/src/swarmhub/storage"

	"github.com/julienschmidt/httprouter"
)

var defaultStart time.Time
var defaultEnd time.Time

func init() {
	dateLayout := "2006-01-02 15:04:05"
	startTimeString := "2000-01-01 00:00:00"
	endTimeString := "2999-01-01 00:00:00"

	var err error
	defaultStart, err = time.Parse(dateLayout, startTimeString)
	if err != nil {
		log.Fatal("Unable to parse startTimeString")
	}

	defaultEnd, err = time.Parse(dateLayout, endTimeString)
	if err != nil {
		log.Fatal("Unable to parse endTimeString")
	}
}

func StartTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testReady, err := validateCanRunTest(ps.ByName("id"))
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Unable to validate test state: %v", err.Error())))
		return
	}

	if testReady == false {
		w.Write([]byte("This test is not in a ready state."))
		return
	}

	var body struct {
		GridID             string
		StartAutomatically bool
		GridRegion         string
	}

	json.NewDecoder(r.Body).Decode(&body)

	gridReady, err := validateCanRunGrid(body.GridID)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Unable to validate grid state: %v", err.Error())))
		return
	}

	if gridReady == false {
		w.Write([]byte("This grid is not in a deployed state."))
		return
	}

	testID := ps.ByName("id")
	scriptID, scriptFilename, err := db.GetScriptFilename(testID)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Unable to get script filename %v", err.Error())))
		return
	}

	locustConfig, err := db.GetLocustConfigByTestId(testID)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Unable to get locust config %v", err.Error())))
		return
	}

	gridID := body.GridID
	gridRegion := body.GridRegion
	gridStartAuto := strconv.FormatBool(body.StartAutomatically)

	params := map[string]string{
		"GRID_NAME":      gridID,
		"GRID_REGION":    gridRegion,
		"GRID_AUTOSTART": gridStartAuto,
		"SCRIPT_ID":      scriptID,
		"LOCUST_COUNT":   fmt.Sprint(locustConfig.Clients),
		"HATCH_RATE":     fmt.Sprint(locustConfig.HatchRate),
		"SCRIPT_KEY":     scriptFilename,
		"DEPLOYMENT":     "true",
	}
	message := &natsMessage{ID: testID, Params: params, DeploymentType: "Test"}
	b, err := json.Marshal(message)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Not publishing nats message. Failed to convert to json: %v", err.Error())))
		return
	}

	err = db.UpdateTestStatus(ps.ByName("id"), "Queued")
	if err != nil {
		fmt.Println("Was unable to update test status! ", err.Error())
		return
	}

	err = sendStartCmd(b)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Was unable to send start command! %v", err.Error())))
		db.UpdateTestStatus(ps.ByName("id"), "Ready")
		return
	}

	err = db.UpdateTestIDinGrid(gridID, testID)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Wasn't able to update Test ID in grid: %v", err.Error())))
		return
	}

	err = db.UpdateGridStatus(gridID, "Deployed")
	if err != nil {
		w.Write([]byte(fmt.Sprintf("Wasn't able to update Grid ID status: %v", err.Error())))
		return
	}

	w.Write([]byte("sent a start command for test id: " + ps.ByName("id")))
}

func UploadTestAttachment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type response struct {
		Status      string
		Description string
	}
	failed := "Failed"
	success := "Success"

	err := r.ParseMultipartForm(50 * 1024 * 1024)
	if err != nil {
		desc := "Failed to read the form, " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	file, fileheader, err := r.FormFile("file")
	if err != nil {
		desc := "No file provided in test attachment upload, " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	testID := ps.ByName("id")
	err = storage.UploadAttachment(testID, fileheader.Filename, file)
	if err != nil {
		desc := "Failed to upload the attachment, " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	err = db.PutTestAttachment(testID, fileheader.Filename)
	if err != nil {
		desc := "Failed to add attachment to the database, " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	desc := "Attachment successfully uploaded!"
	resp := response{success, desc}
	b, _ := json.Marshal(resp)
	w.Write(b)
}

func validateCanRunTest(id string) (bool, error) {
	var test db.Test
	testBytes, err := db.TestByID(id)
	if err != nil {
		fmt.Println("Unable to get test by ID: ", err.Error())
		return false, err
	}

	err = json.Unmarshal(testBytes, &test)
	if err != nil {
		fmt.Println("Error unmarshalling test: ", err.Error())
		return false, err
	}

	if test.Status == "Ready" || test.Status == "Error" {
		return true, nil
	}

	return false, nil
}

func DuplicateTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type result struct {
		Status      string
		TestID      string
		Description string
	}

	testID := ps.ByName("id")

	newTestID, err := db.DuplicateTest(testID)
	if err != nil {
		v := result{Status: "Failed", Description: fmt.Sprintf("Failed to duplicate test %v because %v", testID, err)}
		jsonResult, _ := json.Marshal(v)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResult)
		return
	}

	err = db.DuplicateLocustConfig(testID, newTestID)
	if err != nil {
		db.UpdateTestStatus(newTestID, "Missing info")
		v := result{Status: "Failed", Description: fmt.Sprintf("Failed to duplicate locust config for test %v because %v", testID, err)}
		jsonResult, _ := json.Marshal(v)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResult)
		return
	}

	v := result{Status: "Success", TestID: newTestID}
	jsonResult, _ := json.Marshal(v)
	w.Write(jsonResult)
}

// CancelTestDeployment cancels the test, marks it back as Ready, and cleans up the grid
func CancelTestDeployment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testID := ps.ByName("id")
	gridID, gridRegion, err := db.GetGridByTestID(testID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to send stop command for: " + testID + " " + err.Error()))
		return
	}
	params := map[string]string{
		"GRID_NAME":          gridID,
		"GRID_REGION":        gridRegion,
		"DESTROY_DEPLOYMENT": "true",
	}

	message := &natsMessage{ID: testID, DeploymentType: "Test", Params: params}
	b, err := json.Marshal(message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("Not publishing nats message. Failed to convert to json: ", err.Error())
		return
	}
	err = sendStopCmd(b)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to send stop command for: " + testID))
		return
	}

	// since there is no need to wait for the test cancellation to completely finish
	// go ahead and update the test status so it can be redeployed if need be.
	err = db.UpdateTestStatus(testID, "Ready")
	if err != nil {
		fmt.Println("Was unable to update test status!", err.Error())
	}

	time.Sleep(1 * time.Second) // give some time to help ensure the stop command is run
	err = stopTest(gridID, gridRegion, testID, "CancelTest")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to stopTest within CancelTestDeployment: " + err.Error()))
		return
	}

	w.Write([]byte("sent a stop command for test id: " + ps.ByName("id")))
}

// StopTest calls an ansible job that cleans up the grid and then removes the test from
// the grids database. Status changes are taken care of in the deployer microservice.
func StopTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testID := ps.ByName("id")
	gridID, gridRegion, err := db.GetGridByTestID(testID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to send stop command for: " + ps.ByName("id") + " " + err.Error()))
		return
	}

	if gridID == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Did not get a gridID associated to" + testID))
		return
	}

	err = stopTest(gridID, gridRegion, testID, "StopTest")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to stopTest within StopTest: " + err.Error()))
		return
	}
}

func stopTest(gridID string, gridRegion string, testID string, deploymentType string) error {
	params := map[string]string{
		"GRID_NAME":          gridID,
		"GRID_REGION":        gridRegion,
		"DESTROY_DEPLOYMENT": "true",
	}
	message := &natsMessage{ID: gridID, Params: params, DeploymentType: deploymentType}

	b, err := json.Marshal(message)
	if err != nil {
		err = fmt.Errorf("Not publishing nats message. Failed to convert to json: %v", err.Error())
		return err
	}

	err = sendStartCmd(b)
	if err != nil {
		err = fmt.Errorf("Was unable to send start command! %v", err.Error())
		return err
	}

	err = db.UpdateTestIDinGrid(gridID, "")
	if err != nil {
		err = fmt.Errorf("Wasn't able to update Test ID in grid: %v", err.Error())
		return err
	}

	return nil
}

// PaginateTestInfo is to get the first test, last test, and number of tests based on filters.
func PaginateTestInfo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type paginateInfo struct {
		FirstTest     string
		LastTest      string
		NumberOfTests int
	}

	startDate := extractDateFromURLQuery(r.URL.Query().Get("startdate"), defaultStart)
	endDate := extractDateFromURLQuery(r.URL.Query().Get("enddate"), defaultEnd)
	search := "%" + r.URL.Query().Get("search") + "%"

	firstTest, err := db.GetFirstTest(startDate, endDate, search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lastTest, err := db.GetLastTest(startDate, endDate, search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	numberOfTests, err := db.GetNumberOfTests(startDate, endDate, search)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := paginateInfo{firstTest, lastTest, numberOfTests}
	b, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func GetTestAttachment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testID := ps.ByName("id")
	attachmentID := ps.ByName("attachmentid")
	filename, err := db.GetTestAttachmentName(attachmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buff, err := storage.DownloadAttachment(testID, filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))

	io.Copy(w, bytes.NewReader(buff.Bytes()))
}

func DownloadScriptFiles(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testID := ps.ByName("id")

	scriptID, zipFilename, err := db.GetScriptInfo(testID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buff, err := storage.DownloadScript(scriptID, zipFilename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+zipFilename)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))

	io.Copy(w, bytes.NewReader(buff.Bytes()))
}

func TestsPaginate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testID := ps.ByName("id")
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

	startDate := extractDateFromURLQuery(r.URL.Query().Get("startdate"), defaultStart)
	endDate := extractDateFromURLQuery(r.URL.Query().Get("enddate"), defaultEnd)
	search := "%" + r.URL.Query().Get("search") + "%"

	tests, err := db.TestPaginate(testID, startDate, endDate, search, itemsPerPage)
	if err != nil {
		fmt.Println("Error grabbing data.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(tests)
}

func extractDateFromURLQuery(dateString string, defaultDate time.Time) time.Time {
	if dateString == "" {
		return defaultDate
	}

	dateLayout := "2006-01-02T15:04:05"
	date, err := time.Parse(dateLayout, dateString)
	if err != nil {
		fmt.Printf("Failed to parse date %v, using default %v.\n", dateString, defaultDate)
		return defaultDate
	}

	fmt.Println("Using specified date", date)

	return date
}

func DeleteTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := db.DeleteTestByID(ps.ByName("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func Tests(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	var limit int
	var err error

	limitStr := r.URL.Query().Get("items")
	if limitStr == "" {
		limit = PaginationItems
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			limit = PaginationItems
		}

	}

	startDate := extractDateFromURLQuery(r.URL.Query().Get("startdate"), defaultStart)
	endDate := extractDateFromURLQuery(r.URL.Query().Get("enddate"), defaultEnd)
	search := "%" + r.URL.Query().Get("search") + "%"

	tests, err := db.LatestTests(limit, startDate, endDate, search)
	if err != nil {
		fmt.Println("Error grabbing data.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(tests)
}

func Test(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	ids, ok := r.URL.Query()["id"]

	if !ok || len(ids[0]) < 1 {
		http.Error(w, "Url Param 'id' is missing", http.StatusInternalServerError)
		return
	}

	test, err := db.TestByID(ids[0])
	if err != nil {
		err = fmt.Errorf("failed to make db.TestByID call: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(test)

}

func TestAttachments(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	attachmentFiles, err := db.GetTestAttachments(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(attachmentFiles)
}

func DeleteTestAttachment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	testID := ps.ByName("id")
	attachmentID := ps.ByName("attachmentid")

	attachmentName, err := db.GetTestAttachmentName(attachmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.DeleteTestAttachment(attachmentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = storage.DeleteAttachment(testID, attachmentName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("Successfully deleted attachment!"))
}

func TestFiles(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	id := ps.ByName("id")

	testFiles, err := db.GetTestFiles(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(testFiles)
}

func GetTestPaginateKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

	startDate := extractDateFromURLQuery(r.URL.Query().Get("startdate"), defaultStart)
	endDate := extractDateFromURLQuery(r.URL.Query().Get("enddate"), defaultEnd)
	search := "%" + r.URL.Query().Get("search") + "%"

	testID, err := db.TestGetPaginateKey(id, startDate, endDate, search, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(testID))
}

func LabelToTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	label := ps.ByName("label")
	switch r.Method {
	case http.MethodPost:
		err := db.AddLabelToTest(id, label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodDelete:
		err := db.DeleteLabelFromTest(id, label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Need to use Post or Delete http method", http.StatusMethodNotAllowed)
	}
}

func EditTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	err := r.ParseForm()
	if err != nil {
		fmt.Println("Failed to parse form,", err.Error())
	}

	title := r.Form.Get("Title")
	if title != "" {
		err := db.EditTestTitle(id, title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	desc := r.Form.Get("Desc")
	if desc != "" {
		err := db.EditTestDescription(id, desc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// having a "" value for Result is valid so we aren't using r.Form.Get
	if result, ok := r.Form["Result"]; ok {
		err := db.EditTestResult(id, result[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Write([]byte("Success!"))
}

func CreateTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type response struct {
		Status      string
		Description string
	}
	failed := "Failed"
	success := "Success"
	var t db.Test
	fmt.Println("Starting the process of creating a test.")

	err := r.ParseMultipartForm(50 * 1024 * 1024)
	if err != nil {
		desc := "Failed to read the form, " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	metadata := r.FormValue("metadata")
	if metadata == "" {
		desc := "No metadata provided in test creation"
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	fmt.Println("Metadata section is: ", metadata)
	err = json.Unmarshal([]byte(metadata), &t)
	if err != nil {
		desc := "Unable to unmarshal metadata" + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	file, fileheader, err := r.FormFile("file")
	if err != nil {
		desc := "No file provided in test creation, " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
	}

	if filepath.Ext(fileheader.Filename) != ".zip" {
		desc := "Only Zip Files are supported"
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	zipReader, err := zip.NewReader(file, fileheader.Size)
	if err != nil {
		desc := "Unable to open zip file: " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(b)
		return
	}

	var testFiles db.TestFiles
	var locustFilePresent bool
	testFiles.Name = fileheader.Filename

	for _, zipFile := range zipReader.File {
		fileInfo := db.TestFile{zipFile.Name, zipFile.UncompressedSize, zipFile.Modified}

		// locustfile.py needs to be in the base directory for the zip file to be valid.
		if zipFile.Name == "locustfile.py" {
			locustFilePresent = true
		}
		fmt.Println("Filename in zip:", zipFile.Name)
		testFiles.Files = append(testFiles.Files, fileInfo)
	}

	if locustFilePresent == false {
		desc := ("locustfile.py not present in base directory!")
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(b)
		return
	}

	user := jwt.TokenAudienceFromRequest(r)

	testID, err := db.CreateTest(t, user)
	if err != nil {
		desc := "Failed to commit test to database " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(b)
		return
	}

	scriptID, err := db.CreateTestFiles(testFiles, testID, user)
	if err != nil {
		desc := "Failed to commit files to database " + err.Error()
		fmt.Println(desc)
		resp := response{failed, desc}
		b, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(b)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go storage.UploadScript(testID, scriptID, testFiles.Name, file, &wg)
	wg.Wait()

	// check if test got locust config
	if !db.TestWithLocustConfig(testID) {
		db.UpdateTestStatus(testID, "Missing info")
	}

	desc := "Looks good, sent off to upload!"
	resp := response{success, desc}
	b, _ := json.Marshal(resp)
	w.Write(b)

}

// GetTestStatus returns all the tests with the provided status
func GetTestStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	statusList := r.URL.Query()["status"]
	if len(statusList) == 0 {
		http.Error(w, "please provide a query parameter 'status'", http.StatusBadRequest)
		return
	}

	b, err := db.GetTestsByStatus(statusList)
	if err != nil {
		message := fmt.Sprintf("unsuccessful in getting tests with the following statuses: %v\n err: %v\n", statusList, err)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	w.Write(b)
}

// GetGridStatus gets all grids that have a certain status
func GetGridStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	statusList := r.URL.Query()["status"]
	if len(statusList) == 0 {
		http.Error(w, "please provide a query parameter 'status'", http.StatusBadRequest)
		return
	}

	b, err := db.GetGridsByStatus(statusList...)
	if err != nil {
		message := fmt.Sprintf("unsuccessful in getting tests with the following statuses: %v\n err: %v\n", statusList, err)
		http.Error(w, message, http.StatusBadRequest)
		return
	}

	w.Write(b)
}

// RefreshTestStatus looks at the current tests that are in a deployed state
func RefreshTestStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := db.RefreshTestStatus()
	if err != nil {
		message := fmt.Sprintf("was unable to perform db.RefreshTestStatus: %v\n", err)
		http.Error(w, message, http.StatusBadRequest)
		return
	}
	w.Write([]byte("ok!"))
}
