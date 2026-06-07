package db

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/lib/pq"
)

var database *sql.DB
var dbOpen bool

func OpenDatabase() {
	if dbOpen {
		return
	}
	var err error
	database, err = sql.Open("postgres", "host=/var/run/postgresql dbname=sites user=peter sslmode=disable")
	if err != nil {
		fmt.Println("db open error:", err)
		return
	}
	dbOpen = true
}

type QTH struct {
	CallSign string  `json:"call_sign"`
	Name     string  `json:"name"`
	Lat      float32 `json:"lat"`
	Lon      float32 `json:"lon"`
}

func GetAllQTH() ([]QTH, error) {
	rows, err := database.Query(`SELECT call_sign, COALESCE(name,''), lat, long FROM qth ORDER BY call_sign`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []QTH
	for rows.Next() {
		var q QTH
		if err := rows.Scan(&q.CallSign, &q.Name, &q.Lat, &q.Lon); err != nil {
			return nil, err
		}
		result = append(result, q)
	}
	return result, nil
}

func UpsertQTH(callsign, name, latStr, lonStr string) {
	if !dbOpen {
		return
	}
	lat, err := strconv.ParseFloat(latStr, 32)
	if err != nil {
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 32)
	if err != nil {
		return
	}
	_, err = database.Exec(
		`INSERT INTO qth (call_sign, name, lat, long)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (call_sign) DO UPDATE SET name = $2, lat = $3, long = $4`,
		callsign, name, float32(lat), float32(lon),
	)
	if err != nil {
		fmt.Println("db upsert error:", err)
	}
}

func AddQTH(callsign, name string, lat, lon float32) error {
	_, err := database.Exec(
		`INSERT INTO qth (call_sign, name, lat, long) VALUES ($1, $2, $3, $4)`,
		callsign, name, lat, lon,
	)
	return err
}

func UpdateQTH(callsign, name string, lat, lon float32) error {
	_, err := database.Exec(
		`UPDATE qth SET name=$2, lat=$3, long=$4 WHERE call_sign=$1`,
		callsign, name, lat, lon,
	)
	return err
}

func DeleteQTH(callsign string) error {
	_, err := database.Exec(`DELETE FROM qth WHERE call_sign=$1`, callsign)
	return err
}
