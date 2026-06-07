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
	CallSign string   `json:"call_sign"`
	Name     string   `json:"name"`
	Lat      float32  `json:"lat"`
	Lon      float32  `json:"lon"`
	QNF      float64  `json:"qnf"`
	QNH      *float64 `json:"qnh"`
}

func GetAllQTH() ([]QTH, error) {
	rows, err := database.Query(`SELECT call_sign, COALESCE(name,''), lat, long, COALESCE(qnf, 3), qnh FROM qth ORDER BY call_sign`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []QTH
	for rows.Next() {
		var q QTH
		var qnh sql.NullFloat64
		if err := rows.Scan(&q.CallSign, &q.Name, &q.Lat, &q.Lon, &q.QNF, &qnh); err != nil {
			return nil, err
		}
		if qnh.Valid {
			q.QNH = &qnh.Float64
		}
		result = append(result, q)
	}
	return result, nil
}

func UpsertQTH(callsign, name, latStr, lonStr, altmStr string) {
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
	var qnh *float64
	if altmStr != "" {
		v, err := strconv.ParseFloat(altmStr, 64)
		if err == nil {
			qnh = &v
		}
	}
	_, err = database.Exec(
		`INSERT INTO qth (call_sign, name, lat, long, qnh)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (call_sign) DO UPDATE SET name = $2, lat = $3, long = $4, qnh = COALESCE($5, qth.qnh)`,
		callsign, name, float32(lat), float32(lon), qnh,
	)
	if err != nil {
		fmt.Println("db upsert error:", err)
	}
}

func AddQTH(callsign, name string, lat, lon float32, qnf float64, qnh *float64) error {
	_, err := database.Exec(
		`INSERT INTO qth (call_sign, name, lat, long, qnf, qnh) VALUES ($1, $2, $3, $4, $5, $6)`,
		callsign, name, lat, lon, qnf, qnh,
	)
	return err
}

func UpdateQTH(callsign, name string, lat, lon float32, qnf float64, qnh *float64) error {
	_, err := database.Exec(
		`UPDATE qth SET name=$2, lat=$3, long=$4, qnf=$5, qnh=$6 WHERE call_sign=$1`,
		callsign, name, lat, lon, qnf, qnh,
	)
	return err
}

func DeleteQTH(callsign string) error {
	_, err := database.Exec(`DELETE FROM qth WHERE call_sign=$1`, callsign)
	return err
}

func SetQNHFromTerrain(callsign string, qnh float64) error {
	_, err := database.Exec(`UPDATE qth SET qnh=$2 WHERE call_sign=$1`, callsign, qnh)
	return err
}
