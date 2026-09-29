// Package shortcuts gère la page « Raccourcis » : des boutons configurables qui
// envoient une combinaison de touches, lancent un programme ou ouvrent un
// dossier / une adresse web sur le PC.
package shortcuts

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mickael-pezzoni/deckpad/agent/desktop"
)

type Kind string

const (
	KindKeys    Kind = "keys"    // combinaison de touches, ex. ["win", "shift", "s"]
	KindLaunch  Kind = "launch"  // commande ou programme
	KindOpen    Kind = "open"    // dossier, fichier ou adresse web
	KindCapture Kind = "capture" // capture de tout l'écran, copiée dans le presse-papiers
)

type Shortcut struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Icon    string   `json:"icon"`
	Color   string   `json:"color"`
	Kind    Kind     `json:"kind"`
	Keys    []string `json:"keys,omitempty"`
	Command string   `json:"command,omitempty"`
	Target  string   `json:"target,omitempty"`
}

var (
	ErrNotFound = errors.New("raccourci introuvable")
	ErrInvalid  = errors.New("raccourci invalide")
)

const (
	maxShortcuts = 100
	maxLabel     = 40
	maxText      = 1000
)

// DefaultPath : %APPDATA%\deckpad\shortcuts.json sous Windows, ~/.config/deckpad/shortcuts.json sous Linux.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deckpad", "shortcuts.json"), nil
}

type Store struct {
	path string

	mu   sync.Mutex
	list []Shortcut
}

// Open charge les raccourcis. Sans fichier, on part des raccourcis par défaut
// (le fichier n'est créé qu'à la première modification).
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.list = defaults()
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.list); err != nil {
		return nil, fmt.Errorf("%s illisible : %w", path, err)
	}
	return s, nil
}

func (s *Store) List() []Shortcut {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.list)
}

// Replace remplace toute la liste (ajout, modification, suppression et ordre en
// une fois). Les raccourcis sans identifiant en reçoivent un.
func (s *Store) Replace(list []Shortcut) ([]Shortcut, error) {
	if len(list) > maxShortcuts {
		return nil, fmt.Errorf("%w : %d raccourcis au maximum", ErrInvalid, maxShortcuts)
	}
	seen := map[string]bool{}
	out := make([]Shortcut, len(list))
	for i, sc := range list {
		sc, err := clean(sc)
		if err != nil {
			return nil, err
		}
		if sc.ID == "" || seen[sc.ID] {
			sc.ID = newID()
		}
		seen[sc.ID] = true
		out[i] = sc
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := save(s.path, out); err != nil {
		return nil, err
	}
	s.list = out
	return slices.Clone(out), nil
}

// Run déclenche le raccourci id.
func (s *Store) Run(id string) error {
	s.mu.Lock()
	i := slices.IndexFunc(s.list, func(sc Shortcut) bool { return sc.ID == id })
	var sc Shortcut
	if i >= 0 {
		sc = s.list[i]
	}
	s.mu.Unlock()
	if i < 0 {
		return ErrNotFound
	}
	switch sc.Kind {
	case KindKeys:
		return sendKeys(sc.Keys)
	case KindLaunch:
		return desktop.Launch(func() error { return launch(sc.Command) })
	case KindOpen:
		return desktop.Open(expandHome(sc.Target))
	case KindCapture:
		return capture()
	}
	return ErrInvalid
}

func clean(sc Shortcut) (Shortcut, error) {
	sc.Label = strings.TrimSpace(sc.Label)
	sc.Command = strings.TrimSpace(sc.Command)
	sc.Target = strings.TrimSpace(sc.Target)
	switch {
	case sc.Label == "":
		return sc, fmt.Errorf("%w : nom manquant", ErrInvalid)
	case utf8.RuneCountInString(sc.Label) > maxLabel:
		return sc, fmt.Errorf("%w : nom trop long", ErrInvalid)
	case len(sc.Icon) > maxLabel, len(sc.Color) > maxLabel, len(sc.ID) > maxLabel:
		return sc, ErrInvalid
	}
	switch sc.Kind {
	case KindKeys:
		keys, err := validKeys(sc.Keys)
		if err != nil {
			return sc, err
		}
		sc.Keys, sc.Command, sc.Target = keys, "", ""
	case KindLaunch:
		if sc.Command == "" || len(sc.Command) > maxText {
			return sc, fmt.Errorf("%w : commande manquante", ErrInvalid)
		}
		sc.Keys, sc.Target = nil, ""
	case KindOpen:
		if sc.Target == "" || len(sc.Target) > maxText {
			return sc, fmt.Errorf("%w : dossier ou adresse manquant", ErrInvalid)
		}
		sc.Keys, sc.Command = nil, ""
	case KindCapture:
		sc.Keys, sc.Command, sc.Target = nil, "", ""
	default:
		return sc, fmt.Errorf("%w : type inconnu", ErrInvalid)
	}
	return sc, nil
}

// validKeys accepte des modificateurs puis une seule touche, ex. ctrl+shift+s.
// Les modificateurs sont remis dans un ordre fixe.
func validKeys(keys []string) ([]string, error) {
	var mods []string
	var main string
	for _, k := range keys {
		k = strings.ToLower(strings.TrimSpace(k))
		switch {
		case slices.Contains(Modifiers, k):
			if !slices.Contains(mods, k) {
				mods = append(mods, k)
			}
		case slices.Contains(KeyNames, k) && main == "":
			main = k
		default:
			return nil, fmt.Errorf("%w : touche %q", ErrInvalid, k)
		}
	}
	if main == "" {
		return nil, fmt.Errorf("%w : il manque une touche", ErrInvalid)
	}
	slices.SortFunc(mods, func(a, b string) int { return slices.Index(Modifiers, a) - slices.Index(Modifiers, b) })
	return append(mods, main), nil
}

// expandHome remplace un « ~ » en tête par le dossier personnel.
func expandHome(p string) string {
	if p != "~" && !strings.HasPrefix(p, "~/") && !strings.HasPrefix(p, `~\`) {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, p[1:])
}

func newID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// save écrit le fichier via un fichier temporaire, pour ne jamais le laisser à moitié écrit.
func save(path string, list []Shortcut) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
