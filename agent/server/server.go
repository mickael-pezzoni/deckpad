// Package server expose l'API HTTP et sert l'appli tablette embarquée.
package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/mickael-pezzoni/deckpad/agent/stats"
	"github.com/mickael-pezzoni/deckpad/agent/sysinfo"
	"github.com/mickael-pezzoni/deckpad/agent/webdist"
)

// New construit le routeur : /api/* pour les données, tout le reste pour l'appli.
func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/info", handleInfo)
	mux.Handle("GET /api/stats/stream", streamStats(stats.NewHub(time.Second)))
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

// streamStats pousse une mesure par seconde à la tablette (Server-Sent Events).
func streamStats(hub *stats.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		rc := http.NewResponseController(w)

		ch, unsubscribe := hub.Subscribe()
		defer unsubscribe()
		for {
			select {
			case <-r.Context().Done():
				return
			case s := <-ch:
				data, err := json.Marshal(s)
				if err != nil {
					log.Printf("stats JSON : %v", err)
					continue
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
					return
				}
				if err := rc.Flush(); err != nil {
					return
				}
			}
		}
	}
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
