package db

import (
	"errors"
	"fmt"
)

func CreateGridTemplate(gridTemplate GridTemplate) (GridTemplate, error) {
	sql := `INSERT INTO
				portal.grid_template (name, provider, region, master_type, slave_type, slave_nodes, ttl) 
			VALUES 
				($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`

	var id string
	err := db.QueryRow(sql, gridTemplate.Name, gridTemplate.Provider, gridTemplate.Region, gridTemplate.MasterType,
		gridTemplate.SlaveType, gridTemplate.SlaveNodes, gridTemplate.TTL).Scan(&id)
	if err != nil {
		fmt.Println("error creating grid template: ", err)
		return gridTemplate, err
	}

	gridTemplate.ID = id
	return gridTemplate, nil
}

func GetAllGridTemplates() ([]GridTemplate, error) {
	var gridTemplates []GridTemplate

	sql := `SELECT * 
			FROM 
				portal.grid_template`

	rows, err := db.Query(sql)
	if err != nil {
		fmt.Println("error getting grid templates: ", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		//var id, name, status, ttl, provider, region, master, slave, nodes string
		var gridTemplate GridTemplate
		if err := rows.Scan(&gridTemplate.ID, &gridTemplate.Name, &gridTemplate.Provider, &gridTemplate.Region,
			&gridTemplate.MasterType, &gridTemplate.SlaveType, &gridTemplate.SlaveNodes, &gridTemplate.TTL); err != nil {
			fmt.Println("error parsing grid template: ", err)
			continue
		}
		gridTemplates = append(gridTemplates, gridTemplate)
	}

	return gridTemplates, nil
}

func GetGridTemplateById(id string) (GridTemplate, error) {
	sql := `SELECT * 
			FROM 
				portal.grid_template
	 		WHERE
				id=$1`

	var gridTemplate GridTemplate

	err := db.QueryRow(sql, id).Scan(&gridTemplate.ID, &gridTemplate.Name, &gridTemplate.Provider, &gridTemplate.Region,
		&gridTemplate.MasterType, &gridTemplate.SlaveType, &gridTemplate.SlaveNodes, &gridTemplate.TTL)
	if err != nil {
		fmt.Println("error getting grid template: ", err)
		return gridTemplate, err
	}

	return gridTemplate, nil
}

func UpdateGridTemplate(id string, gridTemplate GridTemplate) error {
	sql := `UPDATE
				portal.grid_template
		  	SET
				name = $1, 
				provider = $2, 
				region = $3,
				master_type = $4,
				slave_type = $5,
				slave_nodes = $6,
				ttl = $7 
		  	WHERE
				id = $8`

	result, err := db.Exec(sql, gridTemplate.Name, gridTemplate.Provider, gridTemplate.Region, gridTemplate.MasterType,
		gridTemplate.SlaveType, gridTemplate.SlaveNodes, gridTemplate.TTL, id)
	if err != nil {
		fmt.Println("error updating grid template: ", err)
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		fmt.Println("error updating grid template: ", err)
		return err
	}

	if count < 1 {
		message := "error updating grid template - grid template not found: " + id
		fmt.Println(message)
		return errors.New(message)
	}

	return nil
}

func DeleteGridTemplate(id string) error {

	sql := `DELETE FROM
				portal.grid_template
 			WHERE
				id = $1`

	_, err := db.Exec(sql, id)
	if err != nil {
		fmt.Println("error deleting grid template: ", err)
		return err
	}

	return nil
}
