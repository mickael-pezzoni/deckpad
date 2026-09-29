package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAppNameUsesPCName(t *testing.T) {
	host := pcName()
	if host == "" {
		t.Skip("nom du PC inconnu")
	}
	dist := fstest.MapFS{
		"manifest.webmanifest": {Data: []byte(`{"name":"deckpad","short_name":"deckpad","start_url":"/"}`)},
		"index.html":           {Data: []byte(`<meta name="apple-mobile-web-app-title" content="deckpad" /><title>deckpad</title>`)},
	}

	rec := httptest.NewRecorder()
	namedManifest(dist)(rec, httptest.NewRequest("GET", "/manifest.webmanifest", nil))
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["short_name"] != host || m["name"] != "deckpad · "+host || m["start_url"] != "/" {
		t.Errorf("manifeste = %v", m)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/manifest+json" {
		t.Errorf("Content-Type = %q", ct)
	}

	rec = httptest.NewRecorder()
	namedIndex(dist)(rec, httptest.NewRequest("GET", "/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `content="`+host+`"`) || !strings.Contains(body, "<title>deckpad · "+host+"</title>") {
		t.Errorf("page = %s", body)
	}
}
