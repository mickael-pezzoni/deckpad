package server

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"math/big"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mickael-pezzoni/deckpad/hub/agents"
	"github.com/mickael-pezzoni/deckpad/hub/auth"
	"github.com/mickael-pezzoni/deckpad/hub/registry"
)

// fakeAgent imite l'API d'un agent : code 123456, clé « secret ».
// Chaque appel a son propre certificat (httptest en partage un par défaut).
func fakeAgent(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/pair/start", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"expires": "bientôt"})
	})
	mux.HandleFunc("POST /api/pair/confirm", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Code, Name string }
		json.NewDecoder(r.Body).Decode(&req)
		if req.Code != "123456" || req.Name != hubName {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"reason":"bad-code","remaining":4}`))
			return
		}
		writeJSON(w, map[string]string{"token": "secret"})
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" || r.Header.Get("Cookie") != "" {
			http.Error(w, "non appairé", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/api/stats/stream" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Write([]byte("data: {\"cpu\":42}\n\n"))
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		writeJSON(w, map[string]string{"path": r.URL.EscapedPath(), "query": r.URL.RawQuery})
	})
	srv := httptest.NewUnstartedServer(mux)
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{selfSigned(t)}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

func selfSigned(t *testing.T) tls.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func register(t *testing.T, reg *registry.Registry, id string, srv *httptest.Server) {
	u, _ := url.Parse(srv.URL)
	host, port, _ := net.SplitHostPort(u.Host)
	p, _ := strconv.Atoi(port)
	reg.Seen(registry.Agent{ID: id, Name: "Gaming", IPs: []string{host}, Port: p})
}

func TestPairAndProxy(t *testing.T) {
	dir := t.TempDir()
	tablets, _ := auth.Open(filepath.Join(dir, "tablets.json"))
	pcs, _ := agents.Open(filepath.Join(dir, "agents.json"))
	reg := registry.New()
	agent := fakeAgent(t)
	register(t, reg, "pc1", agent)

	hub := httptest.NewServer(New(tablets, pcs, reg, nil))
	defer hub.Close()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	do := func(method, path, body string) (*http.Response, string) {
		req, _ := http.NewRequest(method, hub.URL+path, strings.NewReader(body))
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var b strings.Builder
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			b.WriteString(sc.Text())
		}
		return resp, b.String()
	}

	if _, body := do("GET", "/api/agents", ""); body != `[{"id":"pc1","name":"Gaming","online":true,"paired":false}]` {
		t.Fatalf("liste : %s", body)
	}
	if resp, _ := do("GET", "/api/pc/pc1/info", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("relais ouvert sans appairage : %d", resp.StatusCode)
	}
	if resp, _ := do("POST", "/api/pc/pc1/pair/confirm", `{"code":"123456"}`); resp.StatusCode != http.StatusGone {
		t.Fatalf("confirmation sans code demandé : %d", resp.StatusCode)
	}
	if resp, _ := do("POST", "/api/pc/pc1/pair/start", ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("start : %d", resp.StatusCode)
	}
	if resp, body := do("POST", "/api/pc/pc1/pair/confirm", `{"code":"000000"}`); resp.StatusCode != http.StatusForbidden || !strings.Contains(body, `"remaining":4`) {
		t.Fatalf("mauvais code : %d %s", resp.StatusCode, body)
	}
	if resp, _ := do("POST", "/api/pc/pc1/pair/confirm", `{"code":"123456","name":"iPad"}`); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("confirm : %d", resp.StatusCode)
	}
	if _, body := do("GET", "/api/pair/status", ""); body != `{"paired":true}` {
		t.Fatalf("tablette non appairée : %s", body)
	}
	if p, _ := pcs.Get("pc1"); p.Token != "secret" || p.Fingerprint != agents.Fingerprint(agent.Certificate().Raw) {
		t.Fatalf("PC mal enregistré : %+v", p)
	}

	// Relais : chemin (encodage compris) et paramètres gardés, clé ajoutée, cookie retiré.
	resp, body := do("GET", "/api/pc/pc1/shortcuts/a%2Fb/run?x=1", "")
	if resp.StatusCode != http.StatusOK || body != `{"path":"/api/shortcuts/a%2Fb/run","query":"x=1"}` {
		t.Fatalf("relais : %d %s", resp.StatusCode, body)
	}

	// Flux SSE : le premier évènement arrive sans attendre la fin de la réponse.
	req, _ := http.NewRequest("GET", hub.URL+"/api/pc/pc1/stats/stream", nil)
	sresp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	line, _ := bufio.NewReader(sresp.Body).ReadString('\n')
	sresp.Body.Close()
	if line != "data: {\"cpu\":42}\n" {
		t.Fatalf("SSE : %q", line)
	}

	// Un autre PC qui prend l'adresse (autre certificat) est refusé.
	impostor := fakeAgent(t)
	register(t, reg, "pc1", impostor)
	if resp, body := do("GET", "/api/pc/pc1/info", ""); resp.StatusCode != http.StatusBadGateway || !strings.Contains(body, "wrong-pc") {
		t.Fatalf("imposteur accepté : %d %s", resp.StatusCode, body)
	}
}

func TestRevokedAgentIsForgotten(t *testing.T) {
	dir := t.TempDir()
	tablets, _ := auth.Open(filepath.Join(dir, "tablets.json"))
	tok, _ := tablets.Add("iPad")
	pcs, _ := agents.Open(filepath.Join(dir, "agents.json"))
	reg := registry.New()
	agent := fakeAgent(t)
	register(t, reg, "pc1", agent)
	pcs.Set("pc1", agents.Paired{Name: "Gaming", Token: "révoquée", Fingerprint: agents.Fingerprint(agent.Certificate().Raw)})

	h := New(tablets, pcs, reg, nil)
	r := httptest.NewRequest("GET", "/api/pc/pc1/info", nil)
	r.AddCookie(&http.Cookie{Name: tokenCookie, Value: tok})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("statut : %d", w.Code)
	}
	if _, ok := pcs.Get("pc1"); ok {
		t.Fatal("le PC qui a révoqué le hub doit être à réappairer")
	}
}
