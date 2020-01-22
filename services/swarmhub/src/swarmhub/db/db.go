package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lib/pq"
)

var db *sql.DB

type Provider struct {
	Name    string
	Regions []Region
}

type Region struct {
	Name          string
	InstanceTypes []InstanceType
}

type InstanceType struct {
	Name string
}

type Test struct {
	ID          string
	Name        string
	Desc        string
	Status      string
	Labels      []string
	Result      string
	Created     string
	Launched    string
	Stopped     string
	SnapshotURL string
}

type TestFiles struct {
	ID    string
	Name  string
	Files []TestFile
}

type TestFile struct {
	FileName string
	Size     uint32
	Modified time.Time
}

type GridStruct struct {
	ID       string
	Name     string
	Status   string
	TTL      string
	Provider string
	Region   string
	Master   string
	Slave    string
	Nodes    string
}

type GridTemplate struct {
	ID         string `json:"id" db:"id"`
	Name       string `json:"name" db:"name"`
	Provider   string `json:"provider" db:"provider"`
	Region     string `json:"region" db:"region"`
	MasterType string `json:"master_type" db:"master_type"`
	SlaveType  string `json:"slave_type" db:"slave_type"`
	SlaveNodes int    `json:"slave_nodes" db:"slave_nodes"`
	TTL        int    `json:"ttl" db:"ttl"`
}

const dbType = "postgres"

// SourceName is the database source name
var SourceName string

// Set is to connect to the database
func Set() {
	var err error
	db, err = sql.Open(dbType, SourceName)
	if err != nil {
		fmt.Println("error connecting to the database: ", err)
		os.Exit(2)
	}
}

// UpdateTestIDinGrid is used to update the test id that is associated with a grid.
func UpdateTestIDinGrid(gridID string, testID string) error {
	var err error
	sqlString := `UPDATE portal.grid SET test_id = $2 WHERE id = $1`
	if testID == "" {
		_, err = db.Exec(sqlString, gridID, nil)
	} else {
		_, err = db.Exec(sqlString, gridID, testID)
	}
	if err != nil {
		err = fmt.Errorf("unable to update test id in a grid: %v", err)
		return err
	}

	if testID == "" || gridID == "" {
		return nil
	}

	sqlString = `UPDATE portal.test SET grid_id = $2 WHERE id = $1`
	_, err = db.Exec(sqlString, testID, gridID)
	if err != nil {
		err = fmt.Errorf("unable to update grid_id in a test: %v", err)
		return err
	}
	return nil
}

// GetTestByGridID returns the grid assigned to the test, if no grid
// is assigned it will return a blank string with no error
func GetTestByGridID(gridID string) (string, error) {
	sqlString := `SELECT g.test_id FROM portal.grid g WHERE g.id=$1`

	var testID string
	err := db.QueryRow(sqlString, gridID).Scan(&testID)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		err = fmt.Errorf("failed to query row to get grid assigned to test: %v", err)
		return "", err
	}

	return testID, nil
}

// GetGridByTestID returns the grid assigned to the test, if no grid
// is assigned it will return a blank string with no error
func GetGridByTestID(testID string) (string, string, error) {
	sqlString := `SELECT g.id, r.region FROM portal.grid g 
	 INNER JOIN portal.provider_regions r ON g.region_id = r.id 
	 WHERE g.test_id=$1
	 ORDER BY g.created DESC`

	var gridID, gridRegion string
	err := db.QueryRow(sqlString, testID).Scan(&gridID, &gridRegion)
	if err == sql.ErrNoRows {
		return "", "", nil
	} else if err != nil {
		err = fmt.Errorf("failed to query row to get grid assigned to test: %v", err)
		fmt.Println(err)
		return "", "", err
	}

	return gridID, gridRegion, nil
}

func LatestTests(limit int, startDate time.Time, endDate time.Time, search string) ([]byte, error) {
	var b []byte
	rows, err := db.Query(`SELECT t.id, t.name, s.status, r.result, array_agg(l.status), t.description, t.created, t.launched, t.stopped 
		FROM portal.test t 
		INNER JOIN portal.test_status s 
		ON t.status_id = s.id
		LEFT JOIN portal.test_results r
		ON t.result_id = r.id
		LEFT JOIN portal.tests_labels tl
		ON t.id = tl.test_id
		LEFT JOIN portal.labels l
		ON l.id = tl.label_id
		WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
		AND (
			(t.created >= $2 AND t.created <= $3)
			OR (t.launched >= $2 AND t.launched <= $3)
			OR (t.stopped >= $2 AND t.stopped <= $3)
		)
		AND (
			(t.name ILIKE $4)
			OR (t.description ILIKE $4)
			OR (l.status ILIKE $4)
		)
		GROUP BY t.id, t.name, s.status, r.result, t.description, t.created, t.launched, t.stopped 
		ORDER BY t.created DESC LIMIT $1`, limit, startDate, endDate, search)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	var tests []Test

	for rows.Next() {
		var id, name, status, result, desc, created, launched, stopped string
		var sqlCreated, sqlLaunched, sqlStopped pq.NullTime
		var labels []sql.NullString
		var sqlResult sql.NullString
		if err := rows.Scan(&id, &name, &status, &sqlResult, pq.Array(&labels), &desc, &sqlCreated, &sqlLaunched, &sqlStopped); err != nil {
			fmt.Println(err)
			return b, err
		}

		timeFormat := "2006-01-02T15:04:05Z07:00"

		if sqlResult.Valid {
			result = sqlResult.String
		} else {
			result = "-"
		}

		if sqlCreated.Valid {
			created = sqlCreated.Time.Format(timeFormat)
		} else {
			created = "-"
		}

		if sqlLaunched.Valid {
			launched = sqlLaunched.Time.Format(timeFormat)
		} else {
			launched = "-"
		}

		if sqlStopped.Valid {
			stopped = sqlStopped.Time.Format(timeFormat)
		} else {
			stopped = "-"
		}

		tests = append(tests, Test{id, name, desc, status, nullStringToStringSlice(labels), result, created, launched, stopped, ""})
	}

	b, err = json.Marshal(tests)
	if err != nil {
		fmt.Println(err)
	}

	return b, err
}

func CreateTest(test Test, user string) (string, error) {
	var id string
	sql := "INSERT INTO portal.test (name, description, status_id, created_by_user, last_edited_user) VALUES ($1, $2, (SELECT id from portal.test_status WHERE status='Creating'), $3, $3) RETURNING id"

	row := db.QueryRow(sql, test.Name, test.Desc, user)
	err := row.Scan(&id)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return id, err
	}

	fmt.Println("The id from creating the test is:", id)
	return id, err
}

func GetScriptFilename(id string) (string, string, error) {
	sqlString := `SELECT tsi.id, tsi.name FROM portal.test_script_ids AS tsi
	JOIN portal.test AS T ON (t.script_id = tsi.id)
	WHERE t.id=$1`

	var scriptID string
	var scriptFilename string

	err := db.QueryRow(sqlString, id).Scan(&scriptID, &scriptFilename)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	return scriptID, scriptFilename, err
}

// UpdateTestStatusThatUsesGrid makes a call to get the test id associated to the grid id
// it then makes a call to the function that updates test status.
func UpdateTestStatusThatUsesGrid(gridid string, status string) (string, error) {
	query := "SELECT test_id FROM portal.grid WHERE id = $1"

	var testid string
	err := db.QueryRow(query, gridid).Scan(&testid)
	if err == sql.ErrNoRows {
		err = fmt.Errorf("No test associated to grid %v, not updating a correllated test status to %v", gridid, status)
		return testid, err
	} else if err != nil {
		err = fmt.Errorf("failed making query to update test status based on grid: %v", err)
		return testid, err
	}

	err = UpdateTestStatus(testid, status)
	return testid, err
}

// InfoForGrafana returns the data needed by grafana to create a snapshot.
func InfoForGrafana(testID string) (name, gridID string, startTime, endTime time.Time, err error) {
	var sqlGridID sql.NullString
	var sqlStart, sqlEnd pq.NullTime
	query := "SELECT t.name, t.grid_id, t.launched, t.stopped FROM portal.test t WHERE t.id=$1"
	err = db.QueryRow(query, testID).Scan(&name, &sqlGridID, &sqlStart, &sqlEnd)
	if err != nil {
		err = fmt.Errorf("failed to query row: %v", err)
		return name, gridID, startTime, endTime, err
	}

	if sqlGridID.Valid {
		gridID = sqlGridID.String
	}

	if sqlStart.Valid {
		startTime = sqlStart.Time
	} else {
		err = fmt.Errorf("startTime is not valid")
		return name, gridID, startTime, endTime, err
	}

	if sqlEnd.Valid {
		endTime = sqlEnd.Time
	} else {
		fmt.Println("endTime is not valid, using current time")
		endTime = time.Now().UTC()
	}

	return name, gridID, startTime, endTime, err
}

// GrafanaSnapshot saves snapshot links into the database
func GrafanaSnapshot(testID, snapshotURL, snapshotDeleteURL string) error {
	query := "UPDATE portal.test SET (grafana_snapshot_url, grafana_snapshot_delete_url) = ($1, $2) WHERE id = $3"

	_, err := db.Exec(query, snapshotURL, snapshotDeleteURL, testID)
	if err != nil {
		err = fmt.Errorf("database exec failed: %v", err)
		return err
	}

	return nil
}
