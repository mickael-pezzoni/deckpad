// Package auth gère l'appairage avec le hub : un code à 6 chiffres affiché sur
// le PC, échangé contre une clé secrète que le hub garde ensuite.
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
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

const (
	codeTTL     = 5 * time.Minute
	maxAttempts = 5
)

var (
	ErrNoPairing = errors.New("aucun appairage en cours")
	ErrExpired   = errors.New("code expiré")
	ErrLocked    = errors.New("trop d'essais, réessaie dans quelques minutes")
	ErrBadCode   = errors.New("code incorrect")
)

// Device est une tablette appairée. On ne garde que l'empreinte de sa clé :
// lire le fichier ne permet pas de se faire passer pour elle.
type Device struct {
	Name      string    `json:"name"`
	TokenHash string    `json:"tokenHash"`
	PairedAt  time.Time `json:"pairedAt"`
}

type pairing struct {
	code     string
	expires  time.Time
	attempts int
}

type Store struct {
	path  string
	now   func() time.Time
	onNew func(code string) // appelé quand un nouveau code doit être affiché sur le PC

	mu      sync.Mutex
	devices []Device
	pending *pairing
	paired  bool // le dernier code a été accepté
}

// DefaultPath : %APPDATA%\deckpad\devices.json sous Windows, ~/.config/deckpad/devices.json sous Linux.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deckpad", "devices.json"), nil
}

// Open charge les appareils appairés (fichier absent : aucun appareil).
func Open(path string, onNewCode func(code string)) (*Store, error) {
	s := &Store{path: path, now: time.Now, onNew: onNewCode}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.devices); err != nil {
		return nil, fmt.Errorf("%s illisible : %w", path, err)
	}
	return s, nil
}

// Start demande un code. Un code encore valide est réutilisé ; après trop
// d'essais, il faut attendre son expiration (on ne peut pas enchaîner les codes).
func (s *Store) Start() (expires time.Time, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if p := s.pending; p != nil && now.Before(p.expires) {
		if p.attempts >= maxAttempts {
			return time.Time{}, ErrLocked
		}
		return p.expires, nil
	}
	code, err := randomCode()
	if err != nil {
		return time.Time{}, err
	}
	s.pending = &pairing{code: code, expires: now.Add(codeTTL)}
	s.paired = false
	if s.onNew != nil {
		go s.onNew(code)
	}
	return s.pending.expires, nil
}

// CodeState décrit le code pour la fenêtre affichée sur le PC.
type CodeState struct {
	State   string    `json:"state"` // pending, paired, expired, locked ou none
	Code    string    `json:"code,omitempty"`
	Expires time.Time `json:"expires,omitzero"`
}

// Code renvoie l'état du code en cours (le code n'est donné que s'il est encore utilisable).
func (s *Store) Code() CodeState {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pending
	switch {
	case p == nil && s.paired:
		return CodeState{State: "paired"}
	case p == nil:
		return CodeState{State: "none"}
	case !s.now().Before(p.expires):
		return CodeState{State: "expired"}
	case p.attempts >= maxAttempts:
		return CodeState{State: "locked", Expires: p.expires}
	}
	return CodeState{State: "pending", Code: p.code, Expires: p.expires}
}

// Confirm vérifie le code tapé sur la tablette et renvoie sa nouvelle clé.
// remaining indique les essais restants après un code incorrect.
func (s *Store) Confirm(code, deviceName string) (token string, remaining int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.pending
	switch {
	case p == nil:
		return "", 0, ErrNoPairing
	case !s.now().Before(p.expires):
		return "", 0, ErrExpired
	case p.attempts >= maxAttempts:
		return "", 0, ErrLocked
	}
	if subtle.ConstantTimeCompare([]byte(code), []byte(p.code)) != 1 {
		p.attempts++
		if p.attempts >= maxAttempts {
			return "", 0, ErrLocked
		}
		return "", maxAttempts - p.attempts, ErrBadCode
	}

	token, err = randomToken()
	if err != nil {
		return "", 0, err
	}
	// Un appareil qui s'appaire de nouveau (le hub, pour une nouvelle tablette)
	// remplace son ancienne clé au lieu de s'ajouter une fois de plus.
	prev := s.devices
	s.devices = slices.DeleteFunc(slices.Clone(prev), func(d Device) bool { return d.Name == deviceName })
	s.devices = append(s.devices, Device{Name: deviceName, TokenHash: hash(token), PairedAt: s.now()})
	if err := s.save(); err != nil {
		s.devices = prev
		return "", 0, err
	}
	s.pending = nil
	s.paired = true
	return token, 0, nil
}

// Empty indique qu'aucun appareil n'est encore appairé.
func (s *Store) Empty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.devices) == 0
}

// Valid indique si la clé appartient à un appareil appairé.
func (s *Store) Valid(token string) bool {
	if token == "" {
		return false
	}
	h := []byte(hash(token))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.devices {
		if subtle.ConstantTimeCompare(h, []byte(d.TokenHash)) == 1 {
			return true
		}
	}
	return false
}

// save écrit le fichier via un fichier temporaire, pour ne jamais le laisser à moitié écrit.
func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.devices, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
