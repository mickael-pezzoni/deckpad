package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mickael-pezzoni/deckpad/agent/auth"
	"github.com/mickael-pezzoni/deckpad/agent/files"
	"github.com/mickael-pezzoni/deckpad/agent/shortcuts"
)

func TestPairingProtectsAPI(t *testing.T) {
	store, err := auth.Open(filepath.Join(t.TempDir(), "devices.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := shortcuts.Open(filepath.Join(t.TempDir(), "shortcuts.json"))
	if err != nil {
		t.Fatal(err)
	}
	favs, err := files.OpenFavorites(filepath.Join(t.TempDir(), "favorites.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := New(store, keys, favs)
	do := func(method, target, body string, setup func(*http.Request)) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, target, strings.NewReader(body)) // vient de 192.0.2.1 : le hub
		if setup != nil {
			setup(r)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	fromPC := func(r *http.Request) { r.RemoteAddr = "127.0.0.1:5000"; r.Host = "127.0.0.1:8420" }

	if w := do("GET", "/api/info", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("API ouverte sans appairage : %d", w.Code)
	}
	if w := do("POST", "/api/shortcuts/calc/run", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("raccourcis ouverts sans appairage : %d", w.Code)
	}
	if w := do("POST", "/api/pair/start", "", nil); w.Code != http.StatusOK {
		t.Fatalf("start : %d", w.Code)
	}

	// Le code n'est lisible que depuis le PC, jamais via un relais ou un autre nom d'hôte.
	if w := do("GET", "/api/pair/code", "", nil); w.Code != http.StatusForbidden {
		t.Errorf("code lisible depuis le réseau : %d", w.Code)
	}
	if w := do("GET", "/api/pair/code", "", func(r *http.Request) { fromPC(r); r.Header.Set("X-Forwarded-For", "192.168.1.20") }); w.Code != http.StatusForbidden {
		t.Errorf("code lisible via le proxy de dev : %d", w.Code)
	}
	if w := do("GET", "/api/pair/code", "", func(r *http.Request) { fromPC(r); r.Host = "evil.example:8420" }); w.Code != http.StatusForbidden {
		t.Errorf("code lisible via un autre nom d'hôte : %d", w.Code)
	}
	w := do("GET", "/api/pair/code", "", fromPC)
	var st auth.CodeState
	json.NewDecoder(w.Body).Decode(&st)
	if st.State != "pending" || len(st.Code) != 6 {
		t.Fatalf("code attendu depuis le PC : %+v", st)
	}

	wrong := "000000"
	if st.Code == wrong {
		wrong = "111111"
	}
	if w := do("POST", "/api/pair/confirm", `{"code":"`+wrong+`"}`, nil); w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), `"remaining":4`) {
		t.Errorf("mauvais code : %d %s", w.Code, w.Body)
	}
	w = do("POST", "/api/pair/confirm", `{"code":"`+st.Code+`","name":"deckpad hub"}`, nil)
	var res struct{ Token string }
	json.NewDecoder(w.Body).Decode(&res)
	if w.Code != http.StatusOK || res.Token == "" {
		t.Fatalf("confirm : %d %+v", w.Code, res)
	}

	bearer := func(tok string) func(*http.Request) {
		return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+tok) }
	}
	if w := do("POST", "/api/system/nope", "", bearer(res.Token)); w.Code != http.StatusBadRequest {
		t.Errorf("API fermée malgré la clé : %d", w.Code)
	}
	if w := do("POST", "/api/system/nope", "", bearer("fausse")); w.Code != http.StatusUnauthorized {
		t.Errorf("fausse clé acceptée : %d", w.Code)
	}
}
