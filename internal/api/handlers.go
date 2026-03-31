package api

import (
	"encoding/json"
	"net/http"

	"go-radio-streamer/internal/config"
	"go-radio-streamer/internal/streamer"

	"github.com/gorilla/mux"
)

type Router struct {
	*mux.Router
	streamer *streamer.Streamer
	stations []config.Station
}

func NewRouter(s *streamer.Streamer, stations []config.Station) *Router {
	r := &Router{
		Router:   mux.NewRouter(),
		streamer: s,
		stations: stations,
	}
	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	r.HandleFunc("/api/stations", r.handleStations).Methods("GET")
	r.HandleFunc("/api/status", r.handleStatus).Methods("GET")
	r.HandleFunc("/api/play", r.handlePlay).Methods("POST")
	r.HandleFunc("/api/stop", r.handleStop).Methods("POST")
}

func (r *Router) handleStations(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(r.stations)
}

func (r *Router) handleStatus(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(r.streamer.CurrentStatus())
}

func (r *Router) handlePlay(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Station int `json:"station"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if payload.Station < 1 || payload.Station > len(r.stations) {
		http.Error(w, "Invalid station number", http.StatusBadRequest)
		return
	}
	station := r.stations[payload.Station-1]
	if err := r.streamer.Start(station, "239.0.0.1:5004"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (r *Router) handleStop(w http.ResponseWriter, req *http.Request) {
	r.streamer.Stop()
	w.WriteHeader(http.StatusOK)
}
