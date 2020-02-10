package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func UpdateGridStatus(id string, status string) error {

	sql := "UPDATE portal.grid SET status_id = (SELECT id from portal.grid_status WHERE status=$2) WHERE id=$1"

	_, err := db.Exec(sql, id, status)
	if err != nil {
		fmt.Println("Error inserting into database: ", err)
		return err
	}
	return nil
}

func GetGridProviders() ([]byte, error) {

	type jsonStruct struct {
		Providers []string
	}
	var b []byte
	var providerData jsonStruct

	rows, err := db.Query("select name FROM portal.providers ORDER BY (name)")
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		if err := rows.Scan(&provider); err != nil {
			fmt.Println(err)
			return b, err
		}
		providerData.Providers = append(providerData.Providers, provider)
	}

	b, err = json.Marshal(providerData)
	if err != nil {
		fmt.Println(err)
		return b, err
	}

	return b, nil
}

func GetGridRegions(provider string) ([]byte, error) {
	type jsonRegion struct {
		Provider string
		Region   string
	}
	type jsonStruct struct {
		Regions []jsonRegion
	}
	var b []byte
	var regionData jsonStruct

	rows, err := db.Query("select p.name, pr.region FROM portal.provider_regions AS pr JOIN portal.providers AS p ON (p.id = pr.provider) WHERE p.name=$1 ORDER BY (p.name, pr.region)", provider)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		var region string
		if err := rows.Scan(&provider, &region); err != nil {
			fmt.Println(err)
			return b, err
		}
		regionStruct := jsonRegion{Provider: provider, Region: region}
		regionData.Regions = append(regionData.Regions, regionStruct)
	}

	b, err = json.Marshal(regionData)
	if err != nil {
		fmt.Println(err)
		return b, err
	}

	return b, nil
}

func GetGridInstances(provider string, region string) ([]byte, error) {
	type jsonInstance struct {
		Provider string
		Region   string
		Instance string
	}
	type jsonStruct struct {
		Instances []jsonInstance
	}

	var b []byte
	var instanceData jsonStruct

	rows, err := db.Query("select p.name, pr.region, gs.name from portal.region_vm_sizes as gs join portal.provider_regions as pr on (pr.id = gs.provider_region) join portal.providers as p on (p.id = gs.provider) WHERE p.name=$1 AND pr.region=$2 ORDER BY (p.name, pr.region, gs.size)", provider, region)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		var region string
		var instance string
		if err := rows.Scan(&provider, &region, &instance); err != nil {
			fmt.Println(err)
			return b, err
		}
		instanceStruct := jsonInstance{Provider: provider, Region: region, Instance: instance}
		instanceData.Instances = append(instanceData.Instances, instanceStruct)
	}

	b, err = json.Marshal(instanceData)
	if err != nil {
		fmt.Println(err)
		return b, err
	}

	return b, nil
}

func GetAllInstanceTypes() ([]byte, error) {
	var b []byte
	rows, err := db.Query("select p.name, pr.region, gs.name from portal.region_vm_sizes as gs join portal.provider_regions as pr on (pr.id = gs.provider_region) join portal.providers as p on (p.id = gs.provider) ORDER BY (p.name, pr.region, gs.size)")

	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	for rows.Next() {
		var provider, region, instance string
		if err := rows.Scan(&provider, &region, &instance); err != nil {
			fmt.Println(err)
			return b, err
		}
	}

	return b, nil
}

func GetInstanceCore(region, instance_type string) (int, error) {
	row := db.QueryRow("select size from portal.region_vm_sizes where name = $1 AND provider_region = $2", instance_type, region)
	var size int
	if err := row.Scan(&size); err != nil {
		fmt.Println(err)
		return size, err
	}
	return size, nil
}

func CreateGrid(name string, provider string, region string, masterInstance string, slaveInstance string, slaveNumber int, ttl int, user string) {

	sql := `INSERT INTO portal.grid (name, status_id, health_id, created_by_user, last_edited_user, ttl,
		    provider_id, region_id, master_instance_type_id, slave_instance_type_id, nodes) 
			VALUES 
			(
			 $1, 
			 (SELECT id FROM portal.grid_status WHERE status='Ready'), 
			 NULL,
			 $2,$2,$3,
			 (select id FROM portal.providers WHERE name=$4),
			 (select r.id FROM portal.providers p INNER JOIN portal.provider_regions r ON p.id=r.provider WHERE p.name=$4 AND r.region=$5),
			 (select v.id FROM portal.providers p  INNER JOIN portal.provider_regions r ON p.id=r.provider 
				INNER JOIN portal.region_vm_sizes v ON r.id=v.provider_region WHERE p.name=$4 AND r.region=$5 AND v.name=$6),
			 (select v.id FROM portal.providers p  INNER JOIN portal.provider_regions r ON p.id=r.provider 
				INNER JOIN portal.region_vm_sizes v ON r.id=v.provider_region WHERE p.name=$4 AND r.region=$5 AND v.name=$7),
			 $8
			)`

	_, err := db.Exec(sql, name, user, ttl, provider, region, masterInstance, slaveInstance, slaveNumber)
	if err != nil {
		fmt.Println("Error inserting into database for db.CreateGrid: ", err)
	}
}

// GetGridsByStatus returns a list of grids based on status
func GetGridsByStatus(statusList ...string) ([]byte, error) {
	var b []byte
	var where string

	if len(statusList) == 0 {
		fmt.Println("no status provided in list")
		return b, fmt.Errorf("need to provide a status")
	}

	for i := range statusList {
		if i == 0 {
			where = "WHERE (g.status_id = (SELECT id from portal.grid_status WHERE status=$1)"
			continue
		}
		where = where + fmt.Sprintf("\nOR g.status_id = (SELECT id from portal.grid_status WHERE status=$%v)", i+1)
	}
	where = where + ")"

	s := make([]interface{}, len(statusList))
	for i, v := range statusList {
		s[i] = v
	}

	sqlString := `SELECT g.id, g.name, gs.status, g.ttl, p.name, r.region, vm.name, vs.name, nodes FROM portal.grid g 
	 INNER JOIN portal.providers p ON g.provider_id = p.id 
	 INNER JOIN portal.provider_regions r ON g.region_id = r.id 
	 INNER JOIN portal.region_vm_sizes vm on g.master_instance_type_id = vm.id
	 INNER JOIN portal.region_vm_sizes vs on g.slave_instance_type_id = vs.id
	 INNER JOIN portal.grid_status gs on gs.id = g.status_id
	 ` + where + `
	 ORDER BY g.created DESC`

	rows, err := db.Query(sqlString, s...)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	var grids []GridStruct

	for rows.Next() {
		var id, name, status, ttl, provider, region, master, slave, nodes string
		if err := rows.Scan(&id, &name, &status, &ttl, &provider, &region, &master, &slave, &nodes); err != nil {
			fmt.Println(err)
			return b, err
		}

		grids = append(grids, GridStruct{id, name, status, ttl, provider, region, master, slave, nodes})
	}

	if len(grids) == 0 {
		return []byte("[]"), nil
	}

	b, err = json.Marshal(grids)
	if err != nil {
		fmt.Println(err)
	}

	return b, err

}

func GetGrids(limit int) ([]byte, error) {
	var b []byte

	sqlString := `SELECT g.id, g.name, gs.status, g.ttl, p.name, r.region, vm.name, vs.name, nodes FROM portal.grid g 
	 INNER JOIN portal.providers p ON g.provider_id = p.id 
	 INNER JOIN portal.provider_regions r ON g.region_id = r.id 
	 INNER JOIN portal.region_vm_sizes vm on g.master_instance_type_id = vm.id
	 INNER JOIN portal.region_vm_sizes vs on g.slave_instance_type_id = vs.id
	 INNER JOIN portal.grid_status gs on gs.id = g.status_id
	 WHERE g.status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
	 ORDER BY g.created DESC LIMIT $1`

	rows, err := db.Query(sqlString, limit)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	var grids []GridStruct

	for rows.Next() {
		var id, name, status, ttl, provider, region, master, slave, nodes string
		if err := rows.Scan(&id, &name, &status, &ttl, &provider, &region, &master, &slave, &nodes); err != nil {
			fmt.Println(err)
			return b, err
		}

		grids = append(grids, GridStruct{id, name, status, ttl, provider, region, master, slave, nodes})
	}

	b, err = json.Marshal(grids)
	if err != nil {
		fmt.Println(err)
	}

	return b, err

}

// GetGridStatus input the grid id and returns the grid's status
func GetGridStatus(id string) (string, error) {
	var status string

	sqlString := `SELECT gs.status FROM portal.grid g 
	INNER JOIN portal.grid_status gs on gs.id = g.status_id
	WHERE g.id=$1
	ORDER BY g.created DESC`

	err := db.QueryRow(sqlString, id).Scan(&status)
	if err != nil {
		fmt.Println("Getting query failed", err)
		return status, err
	}

	return status, err
}

// GetGridRegion input the grid id and it returns the grid's region
func GetGridRegion(id string) (string, error) {
	var region string
	sqlString := `SELECT gr.region FROM portal.grid g 
	INNER JOIN portal.provider_regions gr on gr.id = g.region_id
	WHERE g.id=$1
	ORDER BY g.created DESC`

	err := db.QueryRow(sqlString, id).Scan(&region)
	if err != nil {
		fmt.Println("Getting query failed", err)
		return region, err
	}

	return region, err
}

// DeleteGridByID should Check the status of the Grid. If a test is deployed on it then special delete process
// would need to be kicked off. Make db status change to the test. If deployed but no tests
// are on it then change TTL tag so it gets deleted. Mark as deleted in DB so it doesn't show up
// anymore in the UI.
//
// current possible grid status: Ready, Error, Deploying, Deployed, Destroyed, Deleted
func DeleteGridByID(id string) error {

	sqlString := "UPDATE portal.grid SET status_id = (SELECT id from portal.grid_status WHERE status='Deleted') WHERE id = $1"
	_, err := db.Exec(sqlString, id)
	if err != nil {
		fmt.Println("Error inserting Deleted grid status into database: ", err)
		return err
	}

	return err

}

func GetGridByID(id string) ([]byte, error) {
	var b []byte

	sqlString := `SELECT g.id, g.name, gs.status, g.ttl, p.name, r.region, vm.name, vs.name, nodes FROM portal.grid g 
	 INNER JOIN portal.providers p ON g.provider_id = p.id 
	 INNER JOIN portal.provider_regions r ON g.region_id = r.id 
	 INNER JOIN portal.region_vm_sizes vm on g.master_instance_type_id = vm.id
	 INNER JOIN portal.region_vm_sizes vs on g.slave_instance_type_id = vs.id
	 INNER JOIN portal.grid_status gs on gs.id = g.status_id
	 WHERE g.id=$1
	 ORDER BY g.created DESC`

	type gridStruct struct {
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

	var grid gridStruct

	err := db.QueryRow(sqlString, id).Scan(&grid.ID, &grid.Name, &grid.Status, &grid.TTL, &grid.Provider, &grid.Region, &grid.Master, &grid.Slave, &grid.Nodes)
	if err != nil {
		fmt.Println(err)
		return b, err
	}

	b, err = json.Marshal(grid)
	if err != nil {
		fmt.Println(err)
	}

	return b, err

}

func GetFirstGrid() (string, error) {

	sqlQuery := `SELECT id FROM portal.grid
	WHERE status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
	ORDER BY created DESC LIMIT 1;`

	var id string
	err := db.QueryRow(sqlQuery).Scan(&id)
	if err == sql.ErrNoRows {
		fmt.Println("No rows for GetFirstGrid.")
		return "", nil
	}
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return id, err
}

func GetLastGrid() (string, error) {
	sqlQuery := `SELECT id FROM portal.grid
	WHERE status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
	ORDER BY created ASC LIMIT 1;`

	var id string
	err := db.QueryRow(sqlQuery).Scan(&id)
	if err == sql.ErrNoRows {
		fmt.Println("No rows for GetLastGrid.")
		return "", nil
	}
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return id, err
}

func GetNumberOfGrids() (int, error) {
	sqlQuery := `SELECT count(id) FROM portal.grid
	WHERE status_id != (SELECT id from portal.grid_status WHERE status='Deleted');`

	var count int
	err := db.QueryRow(sqlQuery).Scan(&count)
	if err == sql.ErrNoRows {
		fmt.Println("No rows for GetNumberOfGrids.")
		return 0, nil
	}
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	return count, err
}

func GridPaginate(testID string, itemsPerPage int) ([]byte, error) {
	var b []byte

	sqlQuery := `SELECT g.id, g.name, gs.status, g.ttl, p.name, r.region, vm.name, vs.name, nodes FROM portal.grid g 
		INNER JOIN portal.providers p ON g.provider_id = p.id 
		INNER JOIN portal.provider_regions r ON g.region_id = r.id 
		INNER JOIN portal.region_vm_sizes vm on g.master_instance_type_id = vm.id
		INNER JOIN portal.region_vm_sizes vs on g.slave_instance_type_id = vs.id
		INNER JOIN portal.grid_status gs on gs.id = g.status_id
		WHERE g.status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
		ORDER BY g.created DESC LIMIT $1 OFFSET (
			SELECT rn FROM (
				Select g.id AS gridid, row_number() over (order by g.created DESC) AS rn FROM portal.grid g
				WHERE g.status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
				ORDER BY g.created DESC
			)
		WHERE gridid = $2) - 1;`

	rows, err := db.Query(sqlQuery, itemsPerPage, testID)
	if err != nil {
		fmt.Println(err)
		return b, err
	}
	defer rows.Close()

	var grids []GridStruct

	for rows.Next() {
		var id, name, status, ttl, provider, region, master, slave, nodes string
		if err := rows.Scan(&id, &name, &status, &ttl, &provider, &region, &master, &slave, &nodes); err != nil {
			fmt.Println(err)
			return b, err
		}

		grids = append(grids, GridStruct{id, name, status, ttl, provider, region, master, slave, nodes})
	}

	b, err = json.Marshal(grids)
	if err != nil {
		fmt.Println(err)
	}

	return b, err
}

func GridGetPaginateKey(testID string, offset int) (string, error) {
	if offset == 0 {
		return testID, nil
	}

	var sqlQuery string
	if offset > 0 {
		sqlQuery = `SELECT id FROM (
		    SELECT id, name, created FROM (
				SELECT id, name, created FROM portal.grid
				WHERE status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
				ORDER BY created DESC LIMIT $1 OFFSET (
					SELECT rn FROM (
						Select id, row_number() over (order by created DESC) AS rn FROM portal.grid
						WHERE status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
						ORDER BY created DESC
					)
				WHERE id = $2)
			) ORDER BY created ASC LIMIT 1);`
	} else {
		sqlQuery = `SELECT id FROM (
		    SELECT id, name, created FROM (
				SELECT id, name, created FROM portal.grid
				WHERE status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
				ORDER BY created ASC LIMIT $1 OFFSET (
					SELECT rn FROM (
						Select id, row_number() over (order by created ASC) AS rn FROM portal.grid
						WHERE status_id != (SELECT id from portal.grid_status WHERE status='Deleted')
						ORDER BY created ASC
					)
				WHERE id = $2)
			) ORDER BY created DESC LIMIT 1);`
		offset = -offset
	}

	var paginateKey string
	err := db.QueryRow(sqlQuery, offset, testID).Scan(&paginateKey)
	if err != nil {
		fmt.Println("Failed to get database query for pagination: ", err)
		return "", err
	}

	return paginateKey, err
}
