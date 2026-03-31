package web

import (
	"net/http"

	"github.com/gorilla/mux"
)

func SetupRoutes(r *mux.Router) {
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("internal/web/static/"))))
	r.HandleFunc("/", handleIndex)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "internal/web/static/index.html")
}