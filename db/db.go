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

func UpsertQTH(callsign, latStr, lonStr string) {
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
		`INSERT INTO qth (call_sign, lat, long)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (call_sign) DO UPDATE SET lat = $2, long = $3`,
		callsign, float32(lat), float32(lon),
	)
	if err != nil {
		fmt.Println("db upsert error:", err)
	}
}
