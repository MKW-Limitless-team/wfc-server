package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"wwfc/database"
)

// HandleMKWTracks handles GET requests to retrieve track frequency data
func HandleMKWTracks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tracks, err := db.GetAllTrackFrequencies()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if tracks == nil {
		tracks = []database.TrackEntry{}
	}

	jsonData, err := json.Marshal(tracks)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonData)))
	w.Write(jsonData)
}
