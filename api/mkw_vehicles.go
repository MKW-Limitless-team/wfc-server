package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// HandleMKWVehicles handles GET requests to retrieve vehicle usage data for a specific player
func HandleMKWVehicles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	profileIDStr := r.URL.Query().Get("profile_id")
	if profileIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	profileID, err := strconv.ParseUint(profileIDStr, 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	vehicleUsage, err := db.GetPlayerVehicleUsage(uint32(profileID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(vehicleUsage)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}
