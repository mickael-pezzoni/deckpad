// Package agents garde les PC appairés avec le hub (clé d'accès et empreinte
// du certificat) et fournit un client HTTPS qui vérifie cette empreinte.
package agents

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Paired est un PC appairé.
type Paired struct {
	Name        string `json:"name"`
	Token       string `json:"token"`       // clé donnée par l'agent (à garder secrète)
	Fingerprint string `json:"fingerprint"` // SHA-256 du certificat de l'agent, en hexadécimal
}

type Store struct {
	path string

	mu     sync.Mutex
	agents map[string]Paired // par identifiant d'agent
}

// Open charge les PC de path (fichier absent : aucun PC).
func Open(path string) (*Store, error) {
	s := &Store{path: path, agents: map[string]Paired{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.agents); err != nil {
		return nil, fmt.Errorf("%s illisible : %w", path, err)
	}
	return s, nil
}

func (s *Store) Get(id string) (Paired, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.agents[id]
	return p, ok
}

// All renvoie une copie des PC appairés.
func (s *Store) All() map[string]Paired {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]Paired, len(s.agents))
	for id, p := range s.agents {
		out[id] = p
	}
	return out
}

func (s *Store) Set(id string, p Paired) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, had := s.agents[id]
	s.agents[id] = p
	if err := s.save(); err != nil {
		if had {
			s.agents[id] = prev
		} else {
			delete(s.agents, id)
		}
		return err
	}
	return nil
}

// Forget oublie un PC (sa clé a été révoquée côté agent).
func (s *Store) Forget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.agents[id]; !ok {
		return nil
	}
	delete(s.agents, id)
	return s.save()
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.agents, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Fingerprint calcule l'empreinte d'un certificat (DER).
func Fingerprint(der []byte) string {
	h := sha256.Sum256(der)
	return hex.EncodeToString(h[:])
}

// ErrWrongPC : le certificat présenté n'est pas celui mémorisé à l'appairage.
var ErrWrongPC = errors.New("certificat inattendu : ce n'est pas le PC appairé")

// Transport parle en HTTPS à un agent. Sans empreinte (appairage en cours), il
// accepte n'importe quel certificat et le note dans seen ; avec, il refuse
// tout autre certificat. Il n'y a pas de CA : c'est l'empreinte qui fait foi.
func Transport(fingerprint string, seen func(fingerprint string)) *http.Transport {
	want, _ := hex.DecodeString(fingerprint)
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, // remplacé par la vérification de l'empreinte ci-dessous
			VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
				if len(raw) == 0 {
					return ErrWrongPC
				}
				got := sha256.Sum256(raw[0])
				if fingerprint != "" && !bytes.Equal(got[:], want) {
					return ErrWrongPC
				}
				if seen != nil {
					seen(hex.EncodeToString(got[:]))
				}
				return nil
			},
		},
		ResponseHeaderTimeout: 15 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
}
