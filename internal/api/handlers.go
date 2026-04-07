package api

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"go-radio-streamer/internal/config"
	"go-radio-streamer/internal/streamer"
	"go-radio-streamer/pkg/aes67"

	"github.com/gorilla/mux"
)

type Router struct {
	*mux.Router
	streamer         *streamer.Streamer
	stations         []config.Station
	multicastAddress string
	refClock         string // full ts-refclk value used in SDP responses
}

func NewRouter(s *streamer.Streamer, stations []config.Station, multicastAddress string, refClock string) *Router {
	r := &Router{
		Router:           mux.NewRouter(),
		streamer:         s,
		stations:         stations,
		multicastAddress: multicastAddress,
		refClock:         refClock,
	}
	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	r.HandleFunc("/api/stations", r.handleStations).Methods("GET")
	r.HandleFunc("/api/status", r.handleStatus).Methods("GET")
	r.HandleFunc("/api/diag", r.handleDiag).Methods("GET")
	r.HandleFunc("/api/stream.sdp", r.handleSDP).Methods("GET")
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

// diagResponse combines live streaming diagnostics with the current SDP so that
// operators can verify the multicast configuration and packet flow in one call.
type diagResponse struct {
	Running       bool   `json:"running"`
	Station       string `json:"station,omitempty"`
	MulticastAddr string `json:"multicast_addr,omitempty"`
	StreamURL     string `json:"stream_url,omitempty"`
	PacketsSent   int64  `json:"packets_sent"`
	BytesSent     int64  `json:"bytes_sent"`
	Underruns     int64  `json:"underruns"`
	LastSentUnix  int64  `json:"last_sent_unix"`
	SDP           string `json:"sdp"`
}

func (r *Router) handleDiag(w http.ResponseWriter, req *http.Request) {
	status := r.streamer.CurrentStatus()
	stats := r.streamer.DiagStats()

	multicastIP, portStr, err := net.SplitHostPort(r.multicastAddress)
	port := 5004
	if err == nil {
		if p, err2 := strconv.Atoi(portStr); err2 == nil && p > 0 {
			port = p
		}
	} else {
		multicastIP = r.multicastAddress
	}

	sessionName := status.Station
	if sessionName == "" {
		sessionName = "gostreamer"
	}

	sdp := aes67.BuildSDP(sessionName, multicastIP, "", port, 97, r.refClock, aes67.DefaultPtimeMs)

	resp := diagResponse{
		Running:       status.Running,
		Station:       status.Station,
		MulticastAddr: r.multicastAddress,
		StreamURL:     stats.StreamURL,
		PacketsSent:   stats.PacketsSent,
		BytesSent:     stats.BytesSent,
		Underruns:     stats.Underruns,
		LastSentUnix:  stats.LastSentUnix,
		SDP:           sdp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (r *Router) handleSDP(w http.ResponseWriter, req *http.Request) {
	status := r.streamer.CurrentStatus()
	sessionName := status.Station
	if sessionName == "" {
		sessionName = "gostreamer"
	}

	multicastIP := req.URL.Query().Get("ip")
	if multicastIP == "" {
		host, _, err := net.SplitHostPort(r.multicastAddress)
		if err == nil {
			multicastIP = host
		} else {
			multicastIP = "239.69.250.171"
		}
	}

	port := 5004
	if v := req.URL.Query().Get("port"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			port = parsed
		}
	} else {
		_, portStr, err := net.SplitHostPort(r.multicastAddress)
		if err == nil {
			if parsed, err := strconv.Atoi(portStr); err == nil && parsed > 0 {
				port = parsed
			}
		}
	}

	payloadType := 97
	if v := req.URL.Query().Get("pt"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed < 128 {
			payloadType = parsed
		}
	}

	// The ?ptp= query parameter accepts either:
	//   - A raw PTP clock ID: "IEEE1588-2008:AA-BB-CC-DD-EE-FF-00-00:0"
	//     (the "ptp=" prefix is added automatically for backward compatibility)
	//   - A full ts-refclk value: "ptp=IEEE1588-2008:…" or "localmac=…"
	// When not provided, the configured reference clock is used as the default.
	tsRefClk := req.URL.Query().Get("ptp")
	if tsRefClk == "" {
		if r.refClock != "" {
			tsRefClk = r.refClock
		} else {
			tsRefClk = aes67.LocalRefClock()
		}
	} else if !strings.HasPrefix(tsRefClk, "ptp=") && !strings.HasPrefix(tsRefClk, "localmac=") {
		// Backward compat: caller passed the raw PTP clock ID without prefix.
		tsRefClk = "ptp=" + tsRefClk
	}

	ptimeMs := aes67.DefaultPtimeMs
	if v := req.URL.Query().Get("ptime"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			ptimeMs = parsed
		}
	}

	originIP := req.URL.Query().Get("originip")

	sdp := aes67.BuildSDP(sessionName, multicastIP, originIP, port, payloadType, tsRefClk, ptimeMs)

	w.Header().Set("Content-Type", "application/sdp")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sdp))
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
	if err := r.streamer.Start(station, r.multicastAddress); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (r *Router) handleStop(w http.ResponseWriter, req *http.Request) {
	r.streamer.Stop()
	w.WriteHeader(http.StatusOK)
}
