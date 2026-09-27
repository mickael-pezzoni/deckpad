// Package server expose l'API HTTP et sert l'appli tablette embarquée.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/mickael-pezzoni/deckpad/agent/auth"
	"github.com/mickael-pezzoni/deckpad/agent/live"
	"github.com/mickael-pezzoni/deckpad/agent/network"
	"github.com/mickael-pezzoni/deckpad/agent/process"
	"github.com/mickael-pezzoni/deckpad/agent/stats"
	"github.com/mickael-pezzoni/deckpad/agent/sysinfo"
	"github.com/mickael-pezzoni/deckpad/agent/system"
	"github.com/mickael-pezzoni/deckpad/agent/webdist"
)

// New construit le routeur : /api/* pour les données, tout le reste pour l'appli.
// Seuls les appareils appairés dans store accèdent à l'API.
func New(store *auth.Store) http.Handler {
	procs := process.NewLister()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/pair/status", handlePairStatus(store))
	mux.HandleFunc("POST /api/pair/start", handlePairStart(store))
	mux.HandleFunc("POST /api/pair/confirm", handlePairConfirm(store))
	mux.HandleFunc("GET /api/pair/code", handlePairCode(store))
	mux.HandleFunc("GET /pair-code", localOnly(handlePairWindow))
	mux.HandleFunc("GET /api/info", handleInfo)
	mux.Handle("GET /api/stats/stream", stream(live.NewHub(time.Second, stats.Collect)))
	mux.Handle("GET /api/processes/stream", stream(live.NewHub(2*time.Second, procs.Apps)))
	mux.HandleFunc("POST /api/processes/kill", handleKill(procs))
	mux.HandleFunc("GET /api/processes/icon", handleIcon(procs))
	mux.Handle("GET /api/network/stream", stream(live.NewHub(time.Second, network.NewMonitor().Collect)))
	mux.HandleFunc("GET /api/network/public-ip", handlePublicIP)
	mux.HandleFunc("POST /api/system/{action}", handleSystem)
	mux.Handle("/", appHandler())
	return requireToken(store, mux)
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	info, err := sysinfo.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, info)
}

func handlePublicIP(w http.ResponseWriter, r *http.Request) {
	ip, err := network.PublicIP(r.Context())
	if err != nil {
		http.Error(w, "IP publique indisponible", http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]string{"ip": ip})
}

func handleIcon(procs *process.Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		icon, err := procs.Icon(r.Context(), r.URL.Query().Get("name"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", icon.ContentType)
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(icon.Data)
	}
}

func handleSystem(w http.ResponseWriter, r *http.Request) {
	action := system.Action(r.PathValue("action"))
	if err := system.Run(action); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("action système : %s", action)
	w.WriteHeader(http.StatusAccepted)
}

func handleKill(procs *process.Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, "nom manquant", http.StatusBadRequest)
			return
		}
		n, err := procs.Kill(r.Context(), req.Name)
		switch {
		case errors.Is(err, process.ErrNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case err != nil:
			http.Error(w, err.Error(), http.StatusForbidden) // souvent : droits admin requis
		default:
			log.Printf("processus %q fermé (%d)", req.Name, n)
			writeJSON(w, map[string]int{"killed": n})
		}
	}
}

// stream pousse chaque mesure du hub à la tablette (Server-Sent Events).
func stream[T any](hub *live.Hub[T]) http.HandlerFunc {
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
			case v := <-ch:
				data, err := json.Marshal(v)
				if err != nil {
					log.Printf("JSON : %v", err)
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
