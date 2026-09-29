// Package auth garde les tablettes autorisées à utiliser le hub. Une tablette
// est ajoutée quand elle tape le code affiché par un PC (voir server/pair.go).
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Tablet est une tablette autorisée. On ne garde que l'empreinte de sa clé :
// lire le fichier ne permet pas de se faire passer pour elle.
type Tablet struct {
	Name      string    `json:"name"`
	TokenHash string    `json:"tokenHash"`
	PairedAt  time.Time `json:"pairedAt"`
}

type Store struct {
	path string

	mu      sync.Mutex
	tablets []Tablet
}

// Open charge les tablettes de path (fichier absent : aucune tablette).
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.tablets); err != nil {
		return nil, fmt.Errorf("%s illisible : %w", path, err)
	}
	return s, nil
}

// Add autorise une nouvelle tablette et renvoie sa clé.
func (s *Store) Add(name string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tablets = append(s.tablets, Tablet{Name: name, TokenHash: hash(token), PairedAt: time.Now()})
	if err := writeJSON(s.path, s.tablets); err != nil {
		s.tablets = s.tablets[:len(s.tablets)-1]
		return "", err
	}
	return token, nil
}

// Valid indique si la clé appartient à une tablette autorisée.
func (s *Store) Valid(token string) bool {
	if token == "" {
		return false
	}
	h := []byte(hash(token))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tablets {
		if subtle.ConstantTimeCompare(h, []byte(t.TokenHash)) == 1 {
			return true
		}
	}
	return false
}

func hash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// writeJSON écrit via un fichier temporaire, pour ne jamais le laisser à moitié écrit.
func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
