package server

import (
	_ "embed"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mickael-pezzoni/deckpad/agent/auth"
)

// token renvoie la clé envoyée par le hub (en-tête « Authorization: Bearer … »).
func token(r *http.Request) string {
	t, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(t)
}

//go:embed pairwin.html
var pairWindowHTML []byte

// requireToken bloque toute l'API au hub non appairé, sauf les routes d'appairage.
func requireToken(store *auth.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/api/pair/") && !store.Valid(token(r)) {
			http.Error(w, "appareil non appairé", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handlePairStart fait apparaître le code sur l'écran du PC.
func handlePairStart(store *auth.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expires, err := store.Start()
		if err != nil {
			pairError(w, err, 0)
			return
		}
		writeJSON(w, map[string]time.Time{"expires": expires})
	}
}

func handlePairConfirm(store *auth.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req); err != nil {
			http.Error(w, "requête invalide", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" || len(name) > 64 {
			name = "deckpad hub"
		}
		token, remaining, err := store.Confirm(req.Code, name)
		if err != nil {
			pairError(w, err, remaining)
			return
		}
		log.Printf("appairage : %q ajouté", name)
		writeJSON(w, map[string]string{"token": token})
	}
}

func pairError(w http.ResponseWriter, err error, remaining int) {
	body := map[string]any{"error": err.Error()}
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, auth.ErrBadCode):
		status, body["reason"], body["remaining"] = http.StatusForbidden, "bad-code", remaining
	case errors.Is(err, auth.ErrLocked):
		status, body["reason"] = http.StatusTooManyRequests, "locked"
	case errors.Is(err, auth.ErrExpired), errors.Is(err, auth.ErrNoPairing):
		status, body["reason"] = http.StatusGone, "expired"
	default:
		log.Printf("appairage : %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// windowPoll : dernière fois (UnixNano) que la fenêtre d'appairage a demandé le code.
var windowPoll atomic.Int64

// PairWindowOpen indique qu'une fenêtre d'appairage est déjà ouverte sur le PC
// (elle interroge le code chaque seconde) : inutile d'en ouvrir une deuxième.
func PairWindowOpen() bool {
	return time.Since(time.Unix(0, windowPoll.Load())) < 3*time.Second
}

// Le code et sa fenêtre ne sont servis qu'au PC lui-même.
func handlePairCode(store *auth.Store) http.HandlerFunc {
	return localOnly(func(w http.ResponseWriter, r *http.Request) {
		windowPoll.Store(time.Now().UnixNano())
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, store.Code())
	})
}

func handlePairWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(pairWindowHTML)
}

// localOnly refuse toute requête qui ne vient pas du PC : adresse distante non locale,
// requête relayée pour la tablette (proxy de dev), ou nom d'hôte inattendu
// (protège d'un site qui ferait pointer son domaine vers 127.0.0.1).
func localOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isLoopback(r.RemoteAddr) || r.Header.Get("X-Forwarded-For") != "" || !localHost(r.Host) {
			http.Error(w, "réservé au PC", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func isLoopback(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func localHost(host string) bool {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host == "localhost" || isLoopback(host)
}
