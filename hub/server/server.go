// Package server expose l'API du hub et sert l'appli tablette : la liste des PC,
// l'appairage, et le relais vers l'API de chaque PC.
package server

import (
	"cmp"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"slices"
	"strconv"
	"sync"

	"github.com/mickael-pezzoni/deckpad/hub/agents"
	"github.com/mickael-pezzoni/deckpad/hub/auth"
	"github.com/mickael-pezzoni/deckpad/hub/registry"
	"github.com/mickael-pezzoni/deckpad/hub/webdist"
)

type Server struct {
	tablets *auth.Store
	agents  *agents.Store
	reg     *registry.Registry
	sec     *Secure // nil : pas de HTTPS
	handoff *handoffs
	app     http.Handler

	mu         sync.Mutex
	pending    map[string]string // appairage en cours : id du PC -> empreinte vue
	transports map[string]*http.Transport
}

// New construit le routeur : /api/* pour les données, tout le reste pour l'appli.
func New(tablets *auth.Store, agentStore *agents.Store, reg *registry.Registry, sec *Secure) http.Handler {
	s := &Server{
		tablets:    tablets,
		agents:     agentStore,
		reg:        reg,
		sec:        sec,
		handoff:    &handoffs{codes: map[string]handoff{}},
		app:        appHandler(),
		pending:    map[string]string{},
		transports: map[string]*http.Transport{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/pair/status", s.handlePairStatus)
	mux.HandleFunc("GET /api/pair/secure", s.handleSecureInfo)
	mux.HandleFunc("POST /api/pair/handoff", s.handleHandoff)
	mux.HandleFunc("POST /api/pair/claim", s.handleClaim)
	mux.HandleFunc("GET /ca", s.handleCA)
	mux.HandleFunc("GET /api/agents", s.handleAgents)
	mux.HandleFunc("POST /api/pc/{id}/pair/start", s.handlePairStart)
	mux.HandleFunc("POST /api/pc/{id}/pair/confirm", s.handlePairConfirm)
	mux.HandleFunc("/api/pc/{id}/", s.handleProxy)
	mux.HandleFunc("/api/", http.NotFound)
	mux.Handle("/", s.app)
	return mux
}

// PC tel que l'appli le voit.
type pcView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Online bool   `json:"online"` // vu sur le réseau
	Paired bool   `json:"paired"` // le hub a sa clé
}

// handleAgents liste les PC vus sur le réseau et ceux déjà appairés (même éteints).
// Publique : la tablette doit pouvoir choisir un PC avant d'être appairée.
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	paired := s.agents.All()
	list := []pcView{}
	for _, a := range s.reg.List() {
		_, ok := paired[a.ID]
		list = append(list, pcView{ID: a.ID, Name: a.Name, Online: true, Paired: ok})
		delete(paired, a.ID)
	}
	for id, p := range paired {
		list = append(list, pcView{ID: id, Name: p.Name, Paired: true})
	}
	slices.SortFunc(list, func(a, b pcView) int { return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID)) })
	writeJSON(w, list)
}

func agentURL(a registry.Agent) string {
	return "https://" + net.JoinHostPort(a.IPs[0], strconv.Itoa(a.Port))
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
	files := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", "/index.html", "/sw.js":
			// Toujours revalider : une nouvelle version du hub doit arriver sur la tablette.
			w.Header().Set("Cache-Control", "no-cache")
		case "/manifest.webmanifest":
			// Type absent de la table de Go : on le donne nous-mêmes.
			w.Header().Set("Content-Type", "application/manifest+json")
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("écriture JSON : %v", err)
	}
}
