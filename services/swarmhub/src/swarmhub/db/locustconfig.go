package db

import (
	"errors"
	"fmt"
)

func CreateLocustConfig(locustConfig LocustConfig) (LocustConfig, error) {
	sql := `INSERT INTO
				portal.locust_config (clients, hatch_rate, test_id) 
			VALUES 
				($1, $2, $3 )
			RETURNING id`

	var id string
	err := db.QueryRow(sql, locustConfig.Clients, locustConfig.HatchRate, locustConfig.TestId).Scan(&id)
	if err != nil {
		fmt.Println("error creating locust config: ", err)
		return locustConfig, err
	}

	locustConfig.ID = id
	return locustConfig, nil
}

func GetLocustConfigByTestId(testId string) (LocustConfig, error) {
	sql := `SELECT * 
			FROM 
				portal.locust_config
	 		WHERE
				test_id=$1`

	var locustConfig LocustConfig

	err := db.QueryRow(sql, testId).Scan(&locustConfig.ID, &locustConfig.Clients, &locustConfig.HatchRate, &locustConfig.TestId)
	if err != nil {
		fmt.Println("error getting locust config: ", err)
		return locustConfig, err
	}

	return locustConfig, nil
}

func UpdateLocustConfig(id string, locustConfig LocustConfig) error {
	sql := `UPDATE
				portal.locust_config
		  	SET
				clients = $1,
				hatch_rate = $2 
		  	WHERE
				id = $3`

	result, err := db.Exec(sql, locustConfig.Clients, locustConfig.HatchRate, id)
	if err != nil {
		fmt.Println("error updating locust config: ", err)
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		fmt.Println("error updating locust config: ", err)
		return err
	}

	if count < 1 {
		message := "error updating locust config - locust config not found: " + id
		fmt.Println(message)
		return errors.New(message)
	}

	return nil
}

func TestWithLocustConfig(testId string) bool {
	sql := `SELECT
				COUNT(*)
			FROM
				portal.locust_config
			WHERE
				test_id = $1`

	var count int
	err := db.QueryRow(sql, testId).Scan(&count)
	if err != nil {
		fmt.Println("error getting locust config info: ", err)
		return false
	}

	if count < 1 {
		fmt.Println("no locust config for test: ", testId)
		return false
	}

	return true
}

func DuplicateLocustConfig(testId string, newTestId string) error {
	locustConfig, err := GetLocustConfigByTestId(testId)
	if err != nil {
		message := fmt.Sprintf("error getting locust config for duplication: " + err.Error())
		fmt.Println(message)
		return errors.New(message)
	}

	var newLocustConfig LocustConfig
	newLocustConfig.Clients = locustConfig.Clients
	newLocustConfig.HatchRate = locustConfig.HatchRate
	newLocustConfig.TestId = newTestId

	_, err = CreateLocustConfig(newLocustConfig)
	if err != nil {
		message := fmt.Sprintf("error duplicating locust config: " + err.Error())
		fmt.Println(message)
		return errors.New(message)
	}

	return nil
}
