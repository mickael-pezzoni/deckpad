package auth

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAddValidReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tablets.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.Add("iPad")
	if err != nil || token == "" {
		t.Fatalf("Add : %q, %v", token, err)
	}
	if !s.Valid(token) || s.Valid("autre") || s.Valid("") {
		t.Error("seule la clé délivrée doit être valide")
	}
	s2, err := Open(path)
	if err != nil || !s2.Valid(token) {
		t.Fatalf("tablette perdue au rechargement : %v", err)
	}
	if strings.Contains(s2.tablets[0].TokenHash, token) {
		t.Error("la clé ne doit pas être stockée en clair")
	}
}
