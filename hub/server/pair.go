package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mickael-pezzoni/deckpad/hub/agents"
)

// Deux cookies distincts : en HTTPS le cookie est « Secure », et le navigateur
// interdit au HTTP de poser un cookie du même nom par-dessus. Avec un seul nom,
// passer une fois en HTTPS cassait l'appairage en HTTP (serveur de dev notamment).
const (
	tokenCookie       = "deckpad_token"
	secureTokenCookie = "deckpad_token_s"
)

// Nom sous lequel le hub apparaît dans les appareils appairés de l'agent.
const hubName = "deckpad hub"

// token renvoie la clé envoyée par la tablette, selon le protocole.
func token(r *http.Request) string {
	if r.TLS != nil {
		if c, err := r.Cookie(secureTokenCookie); err == nil {
			return c.Value
		}
	}
	if c, err := r.Cookie(tokenCookie); err == nil {
		return c.Value
	}
	return ""
}

func setToken(w http.ResponseWriter, r *http.Request, token string) {
	name := tokenCookie
	if r.TLS != nil {
		name = secureTokenCookie
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		MaxAge:   10 * 365 * 24 * 3600, // la tablette reste appairée jusqu'à révocation
		HttpOnly: true,                 // illisible par le JavaScript de la page
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) tabletPaired(r *http.Request) bool {
	return s.tablets.Valid(token(r))
}

func (s *Server) handlePairStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]bool{"paired": s.tabletPaired(r)})
}

// handlePairStart fait afficher un code sur l'écran du PC choisi. Le certificat
// que le PC présente est noté : la confirmation devra se faire avec le même.
//
// Un seul code sert aux deux appairages : il autorise la tablette sur le hub
// (si elle ne l'est pas encore) et donne au hub la clé de ce PC.
func (s *Server) handlePairStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, ok := s.reg.Get(id)
	if !ok {
		reason(w, http.StatusServiceUnavailable, "offline")
		return
	}
	var fp string
	client := &http.Client{Transport: agents.Transport("", func(f string) { fp = f }), Timeout: 10 * time.Second}
	resp, err := client.Post(agentURL(a)+"/api/pair/start", "application/json", nil)
	if err != nil {
		log.Printf("appairage de %s : %v", a.Name, err)
		reason(w, http.StatusServiceUnavailable, "offline")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		s.mu.Lock()
		s.pending[id] = fp
		s.mu.Unlock()
	}
	relay(w, resp)
}

func (s *Server) handlePairConfirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req); err != nil {
		http.Error(w, "requête invalide", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	fp := s.pending[id]
	s.mu.Unlock()
	a, online := s.reg.Get(id)
	switch {
	case !online:
		reason(w, http.StatusServiceUnavailable, "offline")
		return
	case fp == "":
		reason(w, http.StatusGone, "expired") // pas de code demandé (ou hub relancé entre-temps)
		return
	}

	body, _ := json.Marshal(map[string]string{"code": req.Code, "name": hubName})
	client := &http.Client{Transport: agents.Transport(fp, nil), Timeout: 10 * time.Second}
	resp, err := client.Post(agentURL(a)+"/api/pair/confirm", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("appairage de %s : %v", a.Name, err)
		reason(w, http.StatusServiceUnavailable, "offline")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		relay(w, resp) // mauvais code, trop d'essais, code expiré : la tablette l'affiche
		return
	}
	var res struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&res); err != nil || res.Token == "" {
		http.Error(w, "réponse inattendue du PC", http.StatusBadGateway)
		return
	}
	if err := s.agents.Set(id, agents.Paired{Name: a.Name, Token: res.Token, Fingerprint: fp}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	delete(s.pending, id)
	s.mu.Unlock()
	log.Printf("appairage : %s appairé", a.Name)

	if !s.tabletPaired(r) {
		name := strings.TrimSpace(req.Name)
		if name == "" || len(name) > 64 {
			name = "Tablette"
		}
		tok, err := s.tablets.Add(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("appairage : tablette %q ajoutée", name)
		setToken(w, r, tok)
	}
	w.WriteHeader(http.StatusNoContent)
}

// relay renvoie telle quelle une réponse de l'agent (statut et JSON d'erreur).
func relay(w http.ResponseWriter, resp *http.Response) {
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, io.LimitReader(resp.Body, 64<<10))
}

// reason répond une erreur que l'appli sait lire ({"reason": …}).
func reason(w http.ResponseWriter, status int, why string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"reason": why})
}
