package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/mickael-pezzoni/deckpad/hub/tlscert"
)

// Secure décrit le HTTPS du hub, servi à côté du HTTP.
type Secure struct {
	CA      *tlscert.Authority
	TLSPort string // vide si le HTTPS n'a pas pu démarrer
}

const handoffTTL = time.Minute

// handoffs fait passer une tablette appairée du HTTP au HTTPS sans refaire le code :
// l'adresse HTTPS est une autre origine, qui ne voit pas forcément le cookie.
// Un code à usage unique, valable une minute, transporte la clé d'une origine à l'autre.
type handoffs struct {
	mu    sync.Mutex
	codes map[string]handoff
}

type handoff struct {
	token   string
	expires time.Time
}

func (h *handoffs) add(token string) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(b)
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for c, v := range h.codes {
		if now.After(v.expires) {
			delete(h.codes, c)
		}
	}
	h.codes[code] = handoff{token: token, expires: now.Add(handoffTTL)}
	return code, nil
}

func (h *handoffs) take(code string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	v, ok := h.codes[code]
	delete(h.codes, code)
	if !ok || time.Now().After(v.expires) {
		return "", false
	}
	return v.token, true
}

// handleSecureInfo dit à la tablette sur quel port joindre le HTTPS.
func (s *Server) handleSecureInfo(w http.ResponseWriter, r *http.Request) {
	port := ""
	if s.sec != nil {
		port = s.sec.TLSPort
	}
	writeJSON(w, map[string]any{"port": port, "secure": r.TLS != nil})
}

// handleCA sert le certificat à installer sur la tablette (public par nature).
func (s *Server) handleCA(w http.ResponseWriter, r *http.Request) {
	if s.sec == nil || s.sec.CA == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="deckpad-ca.crt"`)
	w.Write(s.sec.CA.CertDER())
}

func (s *Server) handleHandoff(w http.ResponseWriter, r *http.Request) {
	tok := token(r)
	if !s.tablets.Valid(tok) {
		http.Error(w, "tablette non appairée", http.StatusUnauthorized)
		return
	}
	code, err := s.handoff.add(tok)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"code": code})
}

func (s *Server) handleClaim(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	tok, ok := s.handoff.take(req.Code)
	if !ok || !s.tablets.Valid(tok) {
		http.Error(w, "code inconnu ou expiré", http.StatusGone)
		return
	}
	setToken(w, r, tok)
	w.WriteHeader(http.StatusNoContent)
}
