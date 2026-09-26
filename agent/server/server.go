// Package server expose l'API HTTP et sert l'appli tablette embarquée.
package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/mickael-pezzoni/deckpad/agent/sysinfo"
	"github.com/mickael-pezzoni/deckpad/agent/webdist"
)

// New construit le routeur : /api/* pour les données, tout le reste pour l'appli.
func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/info", handleInfo)
	mux.Handle("/", appHandler())
	return mux
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	info, err := sysinfo.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, info)
}

func appHandler() http.Handler {
	dist, err := fs.Sub(webdist.Files, "dist")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Appli web absente : lancez d'abord le build (voir README).", http.StatusNotFound)
		})
	}
	return http.FileServerFS(dist)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("écriture JSON : %v", err)
	}
}
