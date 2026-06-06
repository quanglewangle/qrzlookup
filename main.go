package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/quanglewangle/qrzlook/qrz"
)

func main() {
	username := os.Getenv("QRZ_USERNAME")
	password := os.Getenv("QRZ_PASSWORD")
	port := os.Getenv("QRZ_PORT")

	if username == "" || password == "" {
		log.Fatal("QRZ_USERNAME and QRZ_PASSWORD environment variables are required")
	}
	if port == "" {
		port = "8091"
	}

	client := qrz.NewClient(username, password)

	http.HandleFunc("/lookup/", func(w http.ResponseWriter, r *http.Request) {
		callsign := strings.TrimPrefix(r.URL.Path, "/lookup/")
		callsign = strings.TrimSpace(callsign)
		if callsign == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "callsign required"})
			return
		}

		result, err := client.Lookup(callsign)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	log.Printf("qrzlook listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
