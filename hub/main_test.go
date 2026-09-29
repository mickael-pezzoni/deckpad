package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mickael-pezzoni/deckpad/hub/registry"
)

func TestAgentsAPI(t *testing.T) {
	reg := registry.New()
	h := newHandler(reg)

	get := func() []registry.Agent {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/agents", nil))
		var list []registry.Agent
		if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
			t.Fatalf("réponse %q : %v", w.Body, err)
		}
		return list
	}
	if list := get(); list == nil || len(list) != 0 {
		t.Fatalf("liste vide attendue en JSON [], reçu %v", list)
	}
	reg.Seen(registry.Agent{ID: "a", Name: "Gaming", IPs: []string{"192.168.1.20"}, Port: 8420})
	if list := get(); len(list) != 1 || list[0].Name != "Gaming" {
		t.Fatalf("liste = %+v", list)
	}
}
