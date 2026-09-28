package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/mickael-pezzoni/deckpad/agent/auth"
	"github.com/mickael-pezzoni/deckpad/agent/tlscert"
)

// Secure décrit le HTTPS local, servi à côté du HTTP.
type Secure struct {
	CA       *tlscert.Authority
	HTTPPort string
	TLSPort  string // vide si le HTTPS n'a pas pu démarrer
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
func handleSecureInfo(sec *Secure) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		port := ""
		if sec != nil {
			port = sec.TLSPort
		}
		writeJSON(w, map[string]any{"port": port, "secure": r.TLS != nil})
	}
}

// handleCA sert le certificat à installer sur la tablette (public par nature).
func handleCA(sec *Secure) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if sec == nil || sec.CA == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		w.Header().Set("Content-Disposition", `attachment; filename="deckpad-ca.crt"`)
		w.Write(sec.CA.CertDER())
	}
}

func handleHandoff(store *auth.Store, h *handoffs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := token(r)
		if !store.Valid(tok) {
			http.Error(w, "appareil non appairé", http.StatusUnauthorized)
			return
		}
		code, err := h.add(tok)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"code": code})
	}
}

func handleClaim(store *auth.Store, h *handoffs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req); err != nil {
			http.Error(w, "requête invalide", http.StatusBadRequest)
			return
		}
		token, ok := h.take(req.Code)
		if !ok || !store.Valid(token) {
			http.Error(w, "code inconnu ou expiré", http.StatusGone)
			return
		}
		setToken(w, r, token)
		w.WriteHeader(http.StatusNoContent)
	}
}

// handlePairQR affiche sur le PC un QR code de l'adresse à ouvrir sur la tablette.
func handlePairQR(sec *Secure) http.HandlerFunc {
	return localOnly(func(w http.ResponseWriter, r *http.Request) {
		ip := lanIP()
		if ip == nil || sec == nil {
			http.NotFound(w, r)
			return
		}
		url := "http://" + net.JoinHostPort(ip.String(), sec.HTTPPort)
		png, err := qrcode.Encode(url, qrcode.Medium, 440)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(png)
	})
}

// handlePairURL donne l'adresse encodée dans le QR code, écrite en dessous.
func handlePairURL(sec *Secure) http.HandlerFunc {
	return localOnly(func(w http.ResponseWriter, r *http.Request) {
		url := ""
		if ip := lanIP(); ip != nil && sec != nil {
			url = "http://" + net.JoinHostPort(ip.String(), sec.HTTPPort)
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, map[string]string{"url": url})
	})
}

// lanIP renvoie l'adresse du PC sur le réseau local : celle de l'interface qui
// sert la route par défaut (évite les cartes virtuelles type Hyper-V ou Docker).
// Aucun paquet n'est envoyé : « connecter » un socket UDP choisit juste l'interface.
func lanIP() net.IP {
	if c, err := net.Dial("udp4", "192.0.2.1:9"); err == nil {
		defer c.Close()
		if a, ok := c.LocalAddr().(*net.UDPAddr); ok && !a.IP.IsLoopback() {
			return a.IP
		}
	}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.IsPrivate() && n.IP.To4() != nil {
			return n.IP
		}
	}
	return nil
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
		MaxAge:   10 * 365 * 24 * 3600, // l'appareil reste appairé jusqu'à révocation
		HttpOnly: true,                 // illisible par le JavaScript de la page
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})
}
