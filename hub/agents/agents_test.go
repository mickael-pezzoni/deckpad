package agents

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestTransportPinsFingerprint(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	real := Fingerprint(srv.Certificate().Raw)

	// Appairage : n'importe quel certificat, mais on note son empreinte.
	var seen string
	c := &http.Client{Transport: Transport("", func(f string) { seen = f })}
	if _, err := c.Get(srv.URL); err != nil || seen != real {
		t.Fatalf("premier contact : %v, empreinte %q", err, seen)
	}

	// Ensuite : seul ce certificat passe.
	c = &http.Client{Transport: Transport(real, nil)}
	if _, err := c.Get(srv.URL); err != nil {
		t.Fatalf("bon PC refusé : %v", err)
	}
	other := Fingerprint([]byte("autre"))
	c = &http.Client{Transport: Transport(other, nil)}
	if _, err := c.Get(srv.URL); !errors.Is(err, ErrWrongPC) {
		t.Fatalf("mauvais PC accepté : %v", err)
	}
}

func TestStoreSetForgetReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agents.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Set("a", Paired{Name: "Gaming", Token: "t", Fingerprint: "f"}); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := s2.Get("a"); !ok || p.Token != "t" {
		t.Fatalf("PC perdu au rechargement : %+v", p)
	}
	if err := s2.Forget("a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.Get("a"); ok {
		t.Fatal("PC toujours là après Forget")
	}
}
