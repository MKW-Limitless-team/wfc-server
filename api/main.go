package api

import (
	"net/http"
	"wwfc/common"
	"wwfc/database"
)

var (
	db database.Connection

	apiSecret string
)

func StartServer(reload bool) {
	// Get config
	config := common.GetConfig()

	apiSecret = config.APISecret

	// Start SQL
	db = database.Start(config)

	db.RegisterEvents(config, []string{
		"profile_kicked",
		"profile_banned",
		"profile_unbanned",
	})
}

func Shutdown() {
	db.Close()
}

func RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/api/groups", HandleGroups)
	mux.HandleFunc("/api/stats", HandleStats)
	mux.HandleFunc("/api/pinfo", HandlePinfo)
	mux.HandleFunc("/api/mkw_characters", HandleMKWCharacters)
	mux.HandleFunc("/api/mkw_vehicles", HandleMKWVehicles)
	mux.HandleFunc("/api/mkw_tracks", HandleMKWTracks)
	mux.HandleFunc("/api/mkw_rr", HandleMKWRR)
	mux.HandleFunc("/api/ban", HandleBan)
	mux.HandleFunc("/api/unban", HandleUnban)
	mux.HandleFunc("/api/kick", HandleKick)
}
