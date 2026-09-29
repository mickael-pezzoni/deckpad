package server

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mickael-pezzoni/deckpad/hub/agents"
	"github.com/mickael-pezzoni/deckpad/hub/auth"
	"github.com/mickael-pezzoni/deckpad/hub/registry"
)

// fileAgent imite les routes de fichiers d'un agent : il sert files en
// téléchargement et garde ce qu'il reçoit dans got.
func fileAgent(t *testing.T, files map[string]string, got map[string]string) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/files/download", func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[r.URL.Query().Get("path")]
		if !ok {
			http.Error(w, "introuvable", http.StatusNotFound)
			return
		}
		io.WriteString(w, body)
	})
	mux.HandleFunc("POST /api/files/receive", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if dir := r.URL.Query().Get("dir"); dir == `D:\absent` {
			http.Error(w, "introuvable", http.StatusNotFound)
			return
		} else if dir != "" {
			name = dir + `\` + name
		}
		b, _ := io.ReadAll(r.Body)
		if _, taken := got[name]; taken {
			name = "copie " + name
		}
		got[name] = string(b)
		writeJSON(w, map[string]string{"name": name})
	})
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "non appairé", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{selfSigned(t)}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

func TestTransfer(t *testing.T) {
	dir := t.TempDir()
	tablets, _ := auth.Open(filepath.Join(dir, "tablets.json"))
	tok, _ := tablets.Add("iPad")
	pcs, _ := agents.Open(filepath.Join(dir, "agents.json"))
	reg := registry.New()
	received := map[string]string{}
	src := fileAgent(t, map[string]string{`C:\Users\m\photo.jpg`: "pixels"}, nil)
	dst := fileAgent(t, nil, received)
	for id, srv := range map[string]*httptest.Server{"pc1": src, "pc2": dst} {
		register(t, reg, id, srv)
		pcs.Set(id, agents.Paired{Name: id, Token: "secret", Fingerprint: agents.Fingerprint(srv.Certificate().Raw)})
	}
	hub := New(tablets, pcs, reg, nil)

	send := func(body string) (int, []transferEvent) {
		r := httptest.NewRequest("POST", "/api/transfer", strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: tokenCookie, Value: tok})
		w := httptest.NewRecorder()
		hub.ServeHTTP(w, r)
		var events []transferEvent
		sc := bufio.NewScanner(w.Body)
		for sc.Scan() {
			var e transferEvent
			json.Unmarshal(sc.Bytes(), &e)
			events = append(events, e)
		}
		return w.Code, events
	}
	last := func(events []transferEvent) transferEvent {
		if len(events) == 0 {
			return transferEvent{}
		}
		return events[len(events)-1]
	}

	code, events := send(`{"from":"pc1","to":"pc2","path":"C:\\Users\\m\\photo.jpg"}`)
	if code != http.StatusOK || !last(events).Done || last(events).Name != "photo.jpg" || received["photo.jpg"] != "pixels" {
		t.Fatalf("envoi : %d %+v %v", code, events, received)
	}
	// Déjà présent : c'est l'agent cible qui choisit le nom, le hub le rapporte.
	if _, events := send(`{"from":"pc1","to":"pc2","path":"C:\\Users\\m\\photo.jpg"}`); last(events).Name != "copie photo.jpg" {
		t.Fatalf("renommage : %+v", events)
	}
	if _, events := send(`{"from":"pc1","to":"pc2","path":"C:\\Users\\m\\photo.jpg","dir":"D:\\Photos"}`); last(events).Name != `D:\Photos\photo.jpg` {
		t.Fatalf("dossier choisi : %+v", events)
	}
	if _, events := send(`{"from":"pc1","to":"pc2","path":"C:\\Users\\m\\photo.jpg","dir":"D:\\absent"}`); last(events).Error != "target-folder" {
		t.Fatalf("dossier cible absent : %+v", events)
	}
	if _, events := send(`{"from":"pc1","to":"pc2","path":"C:\\absent.txt"}`); last(events).Error != "not-found" {
		t.Fatalf("fichier absent : %+v", events)
	}
	if code, _ := send(`{"from":"pc1","to":"pc3","path":"C:\\a"}`); code != http.StatusServiceUnavailable {
		t.Fatalf("PC cible inconnu : %d", code)
	}
	if code, _ := send(`{"from":"pc1","to":"pc1","path":"C:\\a"}`); code != http.StatusBadRequest {
		t.Fatalf("même PC : %d", code)
	}

	// Sans tablette appairée, rien ne part.
	r := httptest.NewRequest("POST", "/api/transfer", strings.NewReader(`{"from":"pc1","to":"pc2","path":"x"}`))
	w := httptest.NewRecorder()
	hub.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tablette non appairée : %d", w.Code)
	}
}
