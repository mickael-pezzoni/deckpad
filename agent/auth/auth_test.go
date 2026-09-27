package auth

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newStore(t *testing.T) (*Store, *time.Time, *[]string) {
	t.Helper()
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	var shown []string
	s, err := Open(filepath.Join(t.TempDir(), "deckpad", "devices.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return now }
	s.onNew = func(code string) { shown = append(shown, code) }
	return s, &now, &shown
}

func TestPairingFlow(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	st := s.Code()
	code := st.Code
	if st.State != "pending" || len(code) != 6 {
		t.Fatalf("code attendu, obtenu %q", code)
	}
	token, _, err := s.Confirm(code, "iPad")
	if err != nil || token == "" {
		t.Fatalf("Confirm : %q, %v", token, err)
	}
	if !s.Valid(token) || s.Valid("autre") || s.Valid("") {
		t.Error("seule la clé délivrée doit être valide")
	}
	if st := s.Code(); st.State != "paired" || st.Code != "" {
		t.Errorf("le code doit disparaître après l'appairage : %+v", st)
	}

	// Rechargé depuis le disque : l'appareil est toujours connu, sans clé en clair.
	s2, err := Open(s.path, nil)
	if err != nil || !s2.Valid(token) {
		t.Fatalf("appareil perdu au rechargement : %v", err)
	}
	if strings.Contains(s2.devices[0].TokenHash, token) {
		t.Error("la clé ne doit pas être stockée en clair")
	}
}

func TestStartReusesCodeAndLocks(t *testing.T) {
	s, now, _ := newStore(t)
	s.Start()
	first := s.Code().Code
	s.Start()
	if again := s.Code().Code; again != first {
		t.Error("un code valide doit être réutilisé")
	}

	wrong := "000000"
	if first == wrong {
		wrong = "111111"
	}
	for i := 1; i < maxAttempts; i++ {
		_, remaining, err := s.Confirm(wrong, "x")
		if err != ErrBadCode || remaining != maxAttempts-i {
			t.Fatalf("essai %d : %v, %d restants", i, err, remaining)
		}
	}
	if _, _, err := s.Confirm(wrong, "x"); err != ErrLocked {
		t.Fatalf("attendu ErrLocked, obtenu %v", err)
	}
	if _, _, err := s.Confirm(first, "x"); err != ErrLocked {
		t.Error("même le bon code est refusé une fois bloqué")
	}
	if _, err := s.Start(); err != ErrLocked {
		t.Error("pas de nouveau code tant que le blocage dure")
	}
	if st := s.Code(); st.State != "locked" || st.Code != "" {
		t.Errorf("code bloqué ne doit plus s'afficher : %+v", st)
	}

	*now = now.Add(codeTTL + time.Second)
	if _, err := s.Start(); err != nil {
		t.Errorf("nouveau code attendu après expiration : %v", err)
	}
}

func TestExpiredCode(t *testing.T) {
	s, now, _ := newStore(t)
	s.Start()
	code := s.Code().Code
	*now = now.Add(codeTTL)
	if st := s.Code(); st.State != "expired" {
		t.Errorf("état attendu expired : %+v", st)
	}
	if _, _, err := s.Confirm(code, "x"); err != ErrExpired {
		t.Errorf("attendu ErrExpired, obtenu %v", err)
	}
}
