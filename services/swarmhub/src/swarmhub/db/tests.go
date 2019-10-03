package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

func nullStringToStringSlice(l []sql.NullString) []string {
	var s []string
	for _, v := range l {
		if v.Valid {
			s = append(s, v.String)
		}
	}
	return s
}

// UpdateTestStatus should be used whenever updating a test status. It has special logic to add timestamps
// depending on the status being set.
func UpdateTestStatus(id string, status string) error {

	var sql string

	if status == "Deployed" {
		sql = "UPDATE portal.test SET (status_id, launched) = ((SELECT id from portal.test_status WHERE status=$2), current_timestamp()) WHERE id=$1"
	} else if status == "Stopped" || status == "Expired" {
		sql = "UPDATE portal.test SET (status_id, stopped) = ((SELECT id from portal.test_status WHERE status=$2), current_timestamp())  WHERE id=$1"
	} else {
		sql = "UPDATE portal.test SET status_id = (SELECT id from portal.test_status WHERE status=$2) WHERE id=$1"
	}

	_, err := db.Exec(sql, id, status)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return err
	}

	return nil
}

func TestByID(id string) ([]byte, error) {
	var b []byte
	var err error

	query := `SELECT t.id, t.name, s.status, array_agg(l.status), t.description, t.created, t.launched, t.stopped, t.grafana_snapshot_url 
		FROM portal.test t 
		INNER JOIN portal.test_status s 
		ON t.status_id = s.id
		LEFT JOIN portal.tests_labels tl
	  ON t.id = tl.test_id
	  LEFT JOIN portal.labels l
		ON l.id = tl.label_id
		WHERE t.id=$1
	  GROUP BY t.id, t.name, s.status, t.description, t.created, t.launched, t.stopped, t.grafana_snapshot_url;`

	rows, err := db.Query(query, id)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	var test Test

	for rows.Next() {
		var id, name, status, desc, created, launched, stopped, snapshotURL string
		var sqlCreated, sqlLaunched, sqlStopped pq.NullTime
		var labels []sql.NullString
		if err := rows.Scan(&id, &name, &status, pq.Array(&labels), &desc, &sqlCreated, &sqlLaunched, &sqlStopped, &snapshotURL); err != nil {
			fmt.Println(err)
			return b, err
		}

		timeFormat := "2006-01-02T15:04:05Z07:00"

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
		test = Test{id, name, desc, status, nullStringToStringSlice(labels), "Success", created, launched, stopped, snapshotURL}
	}

	b, err = json.Marshal(test)
	if err != nil {
		fmt.Println(err)
	}

	return b, err
}

// DeleteTestByID Mark the test in the DB as deleted so they do not show up in the UI.
//
// Possibly delete all the data associated to the test and remove from the DB?
//
// Possible states a test can be in: Ready, Creating, Uploading, Queued, Deploying, Deployed,
// Launching, Launched, Running, Stopping, Stopped, Missing info, Upload Failed, Error
func DeleteTestByID(id string) error {
	var status string

	sqlString := `SELECT ts.status FROM portal.test t 
	INNER JOIN portal.test_status ts on ts.id = t.status_id
	WHERE t.id=$1
	ORDER BY t.created DESC`

	err := db.QueryRow(sqlString, id).Scan(&status)
	if err != nil {
		fmt.Println("Getting query failed", err)
		return err
	}

	// TODO: add logic to unprovision the grid if it was deployed or being deployed to.
	if status == "Deployed" || status == "Deploying" || status == "Error" {
		fmt.Println("Need to add logic for Deployed/Deploying before deleting?")
	}

	sqlString = "UPDATE portal.test SET status_id = (SELECT id from portal.test_status WHERE status='Deleted') WHERE id = $1"
	_, err = db.Exec(sqlString, id)
	if err != nil {
		fmt.Println("Error inserting Deleted test status into database: ", err)
		return err
	}

	return err
}

func GetFirstTest(startDate time.Time, endDate time.Time, search string) (string, error) {
	sqlQuery := `Select t.id FROM portal.test t
				INNER JOIN portal.test_status s
				ON t.status_id = s.id
				LEFT JOIN portal.tests_labels tl
				ON t.id = tl.test_id
				LEFT JOIN portal.labels l
				ON l.id = tl.label_id
				WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
				AND (
					(t.created >= $1 AND t.created <= $2)
					OR (t.launched >= $1 AND t.launched <= $2)
					OR (t.stopped >= $1 AND t.stopped <= $2)
				)
				AND (
					(t.name ILIKE $3)
					OR (t.description ILIKE $3)
					OR (l.status ILIKE $3)
				)
				GROUP BY t.id, t.created
				ORDER BY t.created DESC LIMIT 1;`

	var id string
	startDateString := startDate.Format("2006-01-02 15:04:05")
	endDateString := endDate.Format("2006-01-02 15:04:05")
	err := db.QueryRow(sqlQuery, startDateString, endDateString, search).Scan(&id)
	if err == sql.ErrNoRows {
		fmt.Println("No rows for GetFirstTest.")
		return "", nil
	}
	if err != nil {
		err = fmt.Errorf("failed to make query for first test: %v", err)
		fmt.Println(err)
		return "", err
	}
	return id, err
}

func GetLastTest(startDate time.Time, endDate time.Time, search string) (string, error) {

	sqlQuery := `Select t.id FROM portal.test t
				INNER JOIN portal.test_status s
				ON t.status_id = s.id
				LEFT JOIN portal.tests_labels tl
				ON t.id = tl.test_id
				LEFT JOIN portal.labels l
				ON l.id = tl.label_id
				WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
				AND (
					(t.created >= $1 AND t.created <= $2)
					OR (t.launched >= $1 AND t.launched <= $2)
					OR (t.stopped >= $1 AND t.stopped <= $2)
				)
				AND (
					(t.name ILIKE $3)
					OR (t.description ILIKE $3)
					OR (l.status ILIKE $3)
				)
				GROUP BY t.id, t.created
				ORDER BY t.created ASC LIMIT 1;`

	var id string
	startDateString := startDate.Format("2006-01-02 15:04:05")
	endDateString := endDate.Format("2006-01-02 15:04:05")
	err := db.QueryRow(sqlQuery, startDateString, endDateString, search).Scan(&id)
	if err == sql.ErrNoRows {
		fmt.Println("No rows for GetLastTest.")
		return "", nil
	}
	if err != nil {
		err = fmt.Errorf("failed to make query to get last test: %v", err)
		fmt.Println(err)
		return "", err
	}
	return id, err
}

func GetNumberOfTests(startDate time.Time, endDate time.Time, search string) (int, error) {

	sqlQuery := ` SELECT count(id) FROM (
	      SELECT t.id as id FROM portal.test t
				INNER JOIN portal.test_status s
				ON t.status_id = s.id
				LEFT JOIN portal.tests_labels tl
				ON t.id = tl.test_id
				LEFT JOIN portal.labels l
				ON l.id = tl.label_id
				WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
				AND (
					(t.created >= $1 AND t.created <= $2)
					OR (t.launched >= $1 AND t.launched <= $2)
					OR (t.stopped >= $1 AND t.stopped <= $2)
				)
				AND (
					(t.name ILIKE $3)
					OR (t.description ILIKE $3)
					OR (l.status ILIKE $3)
				)
				GROUP BY t.id);`

	var count int
	startDateString := startDate.Format("2006-01-02 15:04:05")
	endDateString := endDate.Format("2006-01-02 15:04:05")
	err := db.QueryRow(sqlQuery, startDateString, endDateString, search).Scan(&count)
	if err == sql.ErrNoRows {
		fmt.Println("No rows for GetNumberOfTests.")
		return 0, nil
	}
	if err != nil {
		err = fmt.Errorf("failed to get total list of tests: %v", err)
		fmt.Println(err)
		return 0, err
	}
	return count, err
}

func TestPaginate(testID string, startDate time.Time, endDate time.Time, search string, itemsPerPage int) ([]byte, error) {
	var b []byte

	sqlQuery := `SELECT t.id, t.name, s.status, r.result, array_agg(l.status), t.created, t.launched, t.stopped FROM portal.test t
		INNER JOIN portal.test_status s
		ON t.status_id = s.id
		LEFT JOIN portal.test_results r
		ON t.result_id = r.id
		LEFT JOIN portal.tests_labels tl
		ON t.id = tl.test_id
		LEFT JOIN portal.labels l
		ON l.id = tl.label_id
		WHERE t.id IN
		(
			SELECT t.id FROM portal.test t
			INNER JOIN portal.test_status s
			ON t.status_id = s.id
			LEFT JOIN portal.tests_labels tl
			ON t.id = tl.test_id
			LEFT JOIN portal.labels l
			ON l.id = tl.label_id
			WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
			AND (
					(t.created >= $3 AND t.created <= $4)
					OR (t.launched >= $3 AND t.launched <= $4) 
					OR (t.stopped >= $3 AND t.stopped <= $4) 
			)
			AND (
				(t.name ILIKE $5)
				OR (t.description ILIKE $5)
				OR (l.status ILIKE $5)
			)
			GROUP BY t.id, t.created
			ORDER BY t.created DESC LIMIT $1 OFFSET
			(
				SELECT rn FROM
				(
					SELECT t.id as id, row_number() over (order by t.created DESC) AS rn, t.created FROM portal.test t
					INNER JOIN portal.test_status s
					ON t.status_id = s.id
					LEFT JOIN portal.tests_labels tl
					ON t.id = tl.test_id
					LEFT JOIN portal.labels l
					ON l.id = tl.label_id
					WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
					AND (
							(t.created >= $3 AND t.created <= $4)
							OR (t.launched >= $3 AND t.launched <= $4) 
							OR (t.stopped >= $3 AND t.stopped <= $4) 
					)
					AND (
						(t.name ILIKE $5)
						OR (t.description ILIKE $5)
						OR (l.status ILIKE $5)
					)
					GROUP BY t.id, t.created
					ORDER BY t.created DESC
				)
			  WHERE id = $2
			) - 1
		)
		GROUP BY t.id, t.name, s.status, r.result, t.created, t.launched, t.stopped
		ORDER BY t.created DESC;`

	startDateString := startDate.Format("2006-01-02 15:04:05")
	endDateString := endDate.Format("2006-01-02 15:04:05")

	rows, err := db.Query(sqlQuery, itemsPerPage, testID, startDateString, endDateString, search)
	if err != nil {
		err = fmt.Errorf("failed to make query for TestPaginate: %v", err)
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	var tests []Test

	for rows.Next() {
		var id, name, status, result, created, launched, stopped string
		var sqlCreated, sqlLaunched, sqlStopped pq.NullTime
		var labels []sql.NullString
		var sqlResult sql.NullString

		if err := rows.Scan(&id, &name, &status, &sqlResult, pq.Array(&labels), &sqlCreated, &sqlLaunched, &sqlStopped); err != nil {
			err = fmt.Errorf("Failed to scan row in TestPaginate: %v", err)
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

		tests = append(tests, Test{id, name, "-", status, nullStringToStringSlice(labels), result, created, launched, stopped, ""})
	}

	b, err = json.Marshal(tests)
	if err != nil {
		err = fmt.Errorf("Failed to marshal json in TestPaginate: %v", err)
		fmt.Println(err)
		return b, err
	}

	return b, err
}

// if an offset of -5 is given, then the test key is given that would allow you to make a call
// for the previous page with an itemcount of 5.
//
// positive offset is typically not used but is provided, not typically used because to go a
// page forward you just need to provide the last item in the current list.
func TestGetPaginateKey(testID string, startDate time.Time, endDate time.Time, search string, offset int) (string, error) {
	if offset == 0 {
		return testID, nil
	}

	var sqlQuery string
	if offset > 0 {
		sqlQuery = `SELECT id FROM (
		    SELECT id, name, created FROM (
				SELECT t.id AS id, t.name AS name, t.created AS created FROM portal.test t
				INNER JOIN portal.test_status s
				ON t.status_id = s.id
				LEFT JOIN portal.tests_labels tl
				ON t.id = tl.test_id
				LEFT JOIN portal.labels l
				ON l.id = tl.label_id
				WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
				AND (
					(t.created >= $3 AND t.created <= $4)
					OR (t.launched >= $3 AND t.launched <= $4)
					OR (t.stopped >= $3 AND t.stopped <= $4)
				)
				AND (
					(t.name ILIKE $5)
					OR (t.description ILIKE $5)
					OR (l.status ILIKE $5)
				)
				GROUP BY t.id, t.created, t.name
				ORDER BY t.created DESC LIMIT $1 OFFSET (
					SELECT rn FROM (
						SELECT t.id as testid, row_number() over (order by t.created DESC) AS rn FROM portal.test t
						INNER JOIN portal.test_status s
						ON t.status_id = s.id
						LEFT JOIN portal.tests_labels tl
						ON t.id = tl.test_id
						LEFT JOIN portal.labels l
						ON l.id = tl.label_id
						WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
						AND (
								(t.created >= $3 AND t.created <= $4)
								OR (t.launched >= $3 AND t.launched <= $4) 
								OR (t.stopped >= $3 AND t.stopped <= $4) 
						)
						AND (
							(t.name ILIKE $5)
							OR (t.description ILIKE $5)
							OR (l.status ILIKE $5)
						)
						GROUP BY t.id, t.created
						ORDER BY t.created DESC
					)
				WHERE testid = $2)
			) ORDER BY created ASC LIMIT 1);`
	} else {
		sqlQuery = `SELECT id FROM (
		    SELECT id, name, created FROM (
				SELECT t.id AS id, t.name AS name, t.created AS created FROM portal.test t
				INNER JOIN portal.test_status s
				ON t.status_id = s.id
				LEFT JOIN portal.tests_labels tl
				ON t.id = tl.test_id
				LEFT JOIN portal.labels l
				ON l.id = tl.label_id
				WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
				AND (
					(t.created >= $3 AND t.created <= $4)
					OR (t.launched >= $3 AND t.launched <= $4)
					OR (t.stopped >= $3 AND t.stopped <= $4)
				)
				AND (
					(t.name ILIKE $5)
					OR (t.description ILIKE $5)
					OR (l.status ILIKE $5)
				)
				GROUP BY t.id, t.created, t.name
				ORDER BY t.created ASC LIMIT $1 OFFSET (
					SELECT rn FROM (
						SELECT t.id AS testid, row_number() over (order by t.created ASC) AS rn FROM portal.test t
						INNER JOIN portal.test_status s
						ON t.status_id = s.id
						LEFT JOIN portal.tests_labels tl
						ON t.id = tl.test_id
						LEFT JOIN portal.labels l
						ON l.id = tl.label_id
						WHERE t.status_id != (SELECT id from portal.test_status WHERE status='Deleted')
						AND (
							(t.created >= $3 AND t.created <= $4)
							OR (t.launched >= $3 AND t.launched <= $4) 
							OR (t.stopped >= $3 AND t.stopped <= $4) 
						)
						AND (
							(t.name ILIKE $5)
							OR (t.description ILIKE $5)
							OR (l.status ILIKE $5)
						)
						GROUP BY t.id, t.created
						ORDER BY t.created ASC
					)
				WHERE testid = $2)
			) ORDER BY created DESC LIMIT 1);`
		offset = -offset
	}

	startDateString := startDate.Format("2006-01-02 15:04:05")
	endDateString := endDate.Format("2006-01-02 15:04:05")

	var paginateKey string
	err := db.QueryRow(sqlQuery, offset, testID, startDateString, endDateString, search).Scan(&paginateKey)
	if err != nil {
		err = fmt.Errorf("failed to get database query for pagination: %v", err)
		fmt.Println(err)
		return "", err
	}

	return paginateKey, err
}

// test file details: filename STRING, filesize INT, last_modified TIMESTAMP,
func GetTestFiles(id string) ([]byte, error) {
	var b []byte

	sql := `SELECT sf.filename, sf.filesize, sf.last_modified 
			FROM portal.script_files AS sf
			JOIN portal.test AS T ON (t.script_id = sf.script_id)
			WHERE t.id=$1`

	rows, err := db.Query(sql, id)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	type testStruct struct {
		Filename     string
		Filesize     int
		LastModified string
	}

	var tests []testStruct

	for rows.Next() {
		var test testStruct
		if err = rows.Scan(&test.Filename, &test.Filesize, &test.LastModified); err != nil {
			fmt.Println(err)
			return b, err
		}
		tests = append(tests, test)
	}

	b, err = json.Marshal(tests)
	if err != nil {
		fmt.Println(err)
		return b, err
	}

	return b, nil
}

func GetTestAttachments(testID string) ([]byte, error) {
	var b []byte

	sql := "SELECT id, filename FROM portal.test_attachments WHERE test_id=$1"
	rows, err := db.Query(sql, testID)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	type attachmentStruct struct {
		ID       string
		Filename string
	}

	var attachments []attachmentStruct

	for rows.Next() {
		var attachment attachmentStruct
		if err = rows.Scan(&attachment.ID, &attachment.Filename); err != nil {
			fmt.Println(err)
			return b, err
		}
		attachments = append(attachments, attachment)
	}

	b, err = json.Marshal(attachments)
	if err != nil {
		fmt.Println(err)
		return b, err
	}

	return b, nil
}

func GetTestAttachmentName(ID string) (string, error) {
	var filename string

	sql := "SELECT filename FROM portal.test_attachments WHERE id=$1"

	err := db.QueryRow(sql, ID).Scan(&filename)
	if err != nil {
		fmt.Println("Querying for test attachment name failed", err)
		return filename, err
	}

	return filename, nil
}

func GetScriptInfo(testID string) (string, string, error) {
	var filename string
	var scriptID string

	sql := "SELECT s.id, s.name FROM portal.test t INNER JOIN portal.test_script_ids s ON t.script_id = s.id WHERE t.id=$1"

	err := db.QueryRow(sql, testID).Scan(&scriptID, &filename)
	if err != nil {
		fmt.Println("Querying for test attachment name failed", err)
		return scriptID, filename, err
	}

	return scriptID, filename, nil
}

func DeleteTestAttachment(ID string) error {
	sql := "DELETE FROM portal.test_attachments WHERE id=$1"

	_, err := db.Exec(sql, ID)
	if err != nil {
		fmt.Println("Failed in deleting test attachment.", err)
		return err
	}

	return nil
}

func PutTestAttachment(testID string, filename string) error {
	sql := "INSERT INTO portal.test_attachments (test_id, filename) VALUES ($1, $2)"

	_, err := db.Exec(sql, testID, filename)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return err
	}

	return nil
}

func CreateTestFiles(testFiles TestFiles, id string, user string) (string, error) {
	var scriptID string

	sql := "INSERT INTO portal.test_script_ids (name) VALUES ($1) RETURNING id"

	rows, err := db.Query(sql, testFiles.Name)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&scriptID); err != nil {
			return "", err
		}
	}

	sqlUpdate := "UPDATE portal.test SET script_id = $1 WHERE id = $2"

	_, err = db.Query(sqlUpdate, scriptID, id)
	if err != nil {
		fmt.Println("Error updating test to have script ID: ", err)
		return "", err
	}

	vals := []interface{}{}
	vals = append(vals, scriptID, user)
	sqlStr := "INSERT INTO portal.script_files (filename, filesize, last_modified, script_id, last_edited_user) VALUES "
	inserts := []string{}

	for j, file := range testFiles.Files {
		i := 3 + j*3
		inserts = append(inserts, fmt.Sprintf("($%v, $%v, $%v, $%v, $%v)", i, i+1, i+2, 1, 2))
		vals = append(vals, file.FileName, file.Size, file.Modified)
	}
	sqlStr = sqlStr + strings.Join(inserts, ", ")

	rows, err = db.Query(sqlStr, vals...)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return "", err
	}
	defer rows.Close()

	return scriptID, err
}

func EditTestTitle(id string, title string) error {

	sql := "UPDATE portal.test SET name = $2 WHERE id = $1"

	_, err := db.Exec(sql, id, title)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return err
	}

	return nil
}

// EditTestResult will edit the test to a specified result
func EditTestResult(id string, result string) error {
	var err error
	if result == "" {
		sql := "UPDATE portal.test SET result_id=$2 WHERE id=$1"
		_, err = db.Exec(sql, id, nil)
	} else {
		sql := "UPDATE portal.test SET result_id = (SELECT id from portal.test_results WHERE result=$2) WHERE id=$1"
		_, err = db.Exec(sql, id, result)
	}

	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return err
	}

	return nil
}

func EditTestDescription(id string, description string) error {
	sql := "UPDATE portal.test SET description = $2 WHERE id = $1"

	_, err := db.Exec(sql, id, description)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return err
	}

	return nil
}

func DeleteTest(id string) {
	sql := "INSERT INTO portal.test (name, description, created) VALUES ($1)"

	_, err := db.Exec(sql, id)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
	}
}

func DuplicateTest(testID string) (string, error) {
	var newID string
	sql := `INSERT INTO portal.test (name, description, script_id, created_by_user, last_edited_user, status_id) 
    SELECT concat('COPY of ', t.name), t.description, t.script_id, t.created_by_user, t.last_edited_user, s.id
	FROM (SELECT name, description, script_id, status_id, created_by_user, last_edited_user FROM portal.test WHERE id=$1) as t
	CROSS JOIN
	(SELECT id from portal.test_status WHERE status='Ready') as s
	RETURNING id;`

	row := db.QueryRow(sql, testID)
	err := row.Scan(&newID)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return newID, err
	}

	fmt.Println("The id from creating the test is:", newID)

	return newID, err
}

func addNewLabel(label string) (int, error) {
	sqlNewLabel := "INSERT INTO portal.labels (status) VALUES (lower($1)) RETURNING id"
	var labelID int
	err := db.QueryRow(sqlNewLabel, label).Scan(&labelID)
	if err != nil {
		err = fmt.Errorf("Failed to insert a new label: %v", err)
		return labelID, err
	}

	return labelID, nil
}

func AddLabelToTest(testID string, label string) error {
	sqlGetLabel := "SELECT id FROM portal.labels WHERE status = lower($1)"
	var labelID int
	err := db.QueryRow(sqlGetLabel, label).Scan(&labelID)
	if err == sql.ErrNoRows {
		var err2 error
		labelID, err2 = addNewLabel(label)
		if err2 != nil {
			err2 = fmt.Errorf("Failed to add new label: %v", err2)
			return err2
		}
	} else if err != nil {
		err = fmt.Errorf("Failed to make query for label: %v", err)
		return err
	}

	sqlAddLabelToTest := "INSERT INTO portal.tests_labels (test_id, label_id) VALUES ($1, $2)"

	_, err = db.Exec(sqlAddLabelToTest, testID, labelID)
	if err != nil {
		err = fmt.Errorf("Unable to add label to test: %v", err)
		return err
	}

	return nil
}

func DeleteLabelFromTest(testID string, label string) error {
	sqlDeleteLabelFromTest := `DELETE FROM portal.tests_labels tl 
		 WHERE tl.test_id = $1 
		 AND tl.label_id = (SELECT id from portal.labels WHERE status=$2)`
	_, err := db.Exec(sqlDeleteLabelFromTest, testID, label)
	if err != nil {
		err = fmt.Errorf("Error deleting label from test: %v", err)
		return err
	}
	return nil
}

// GetTestsByStatus get all tests that have a status within the list
func GetTestsByStatus(statusList []string) ([]byte, error) {
	var b []byte
	var where string
	if len(statusList) == 0 {
		fmt.Println("no status provided in list")
		return b, fmt.Errorf("need to provide a status")
	}

	for i := range statusList {
		if i == 0 {
			where = "WHERE (t.status_id = (SELECT id from portal.test_status WHERE status=$1)"
			continue
		}
		where = where + fmt.Sprintf("\nOR t.status_id = (SELECT id from portal.test_status WHERE status=$%v)", i+1)
	}
	where = where + ")"

	query := `SELECT t.id, t.name, s.status, r.result, array_agg(l.status), t.description, t.created, t.launched, t.stopped 
	FROM portal.test t 
	INNER JOIN portal.test_status s 
	ON t.status_id = s.id
	LEFT JOIN portal.test_results r
	ON t.result_id = r.id
	LEFT JOIN portal.tests_labels tl
	ON t.id = tl.test_id
	LEFT JOIN portal.labels l
	ON l.id = tl.label_id
	` + where + `
	GROUP BY t.id, t.name, s.status, r.result, t.description, t.created, t.launched, t.stopped 
	ORDER BY t.created DESC
	`

	s := make([]interface{}, len(statusList))
	for i, v := range statusList {
		s[i] = v
	}

	rows, err := db.Query(query, s...)
	if err != nil {
		fmt.Println("received err:", err)
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
			fmt.Println("error scanning:", err)
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

	if len(tests) == 0 {
		return []byte("[]"), nil
	}

	b, err = json.Marshal(tests)
	if err != nil {
		fmt.Println("marshal err:", err)
	}

	return b, err
}

// RefreshTestStatus is used to make sure the test are in the correct status
func RefreshTestStatus() error {
	query := `UPDATE portal.test SET status_id = (SELECT id from portal.test_status WHERE status='Stopped')
	          WHERE id IN (
							SELECT t.id FROM portal.test t
							INNER JOIN portal.test_status s 
							ON t.status_id = s.id
							WHERE t.status_id = (SELECT id from portal.test_status WHERE status='Deployed')
							AND t.id NOT IN (
								SELECT g.test_id FROM portal.grid g 
								INNER JOIN portal.grid_status gs
								ON g.status_id = gs.id
								where g.status_id = (SELECT id FROM portal.grid_status WHERE status='Deployed')
							)
						)`
	_, err := db.Exec(query)
	return err
}
