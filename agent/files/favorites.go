package files

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// Favorite est un fichier ou un dossier mis en favori depuis la tablette.
type Favorite struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Folder string `json:"folder"` // dossier que la tablette ouvre : celui qui contient le fichier, ou le dossier lui-même
	Dir    bool   `json:"dir"`
	Size   uint64 `json:"size"` // fichiers seulement
}

const maxFavorites = 50

var ErrTooMany = fmt.Errorf("%d favoris au maximum", maxFavorites)

// Favorites garde la liste des favoris dans favorites.json, à côté de devices.json.
type Favorites struct {
	path string

	mu    sync.Mutex
	paths []string // du plus récent au plus ancien
}

// stored est le contenu de favorites.json. Home dit que le dossier utilisateur
// a déjà été ajouté une fois : retiré ensuite, il ne revient pas.
type stored struct {
	Paths []string `json:"paths"`
	Home  bool     `json:"home"`
}

// homeDir : le dossier utilisateur, remplacé dans les tests.
var homeDir = os.UserHomeDir

func DefaultFavoritesPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deckpad", "favorites.json"), nil
}

// OpenFavorites charge les favoris. La première fois, le dossier utilisateur
// (C:\Users\nom, /home/nom) y est ajouté.
func OpenFavorites(path string) (*Favorites, error) {
	f := &Favorites{path: path}
	var st stored
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	case len(b) > 0 && b[0] == '[': // ancien format : la liste seule
		if err := json.Unmarshal(b, &st.Paths); err != nil {
			return nil, fmt.Errorf("%s illisible : %w", path, err)
		}
	default:
		if err := json.Unmarshal(b, &st); err != nil {
			return nil, fmt.Errorf("%s illisible : %w", path, err)
		}
	}
	f.paths = st.Paths
	if st.Home {
		return f, nil
	}
	if home, err := homeDir(); err == nil && home != "" {
		home = filepath.Clean(home)
		if !slices.ContainsFunc(f.paths, func(p string) bool { return pathKey(p) == pathKey(home) }) {
			f.paths = append(f.paths, home)
		}
	}
	if err := f.save(f.paths); err != nil {
		return nil, err
	}
	return f, nil
}

// List renvoie les favoris présents sur un disque visible. Les autres restent
// en mémoire : une clé USB débranchée peut revenir.
func (f *Favorites) List(ctx context.Context) ([]Favorite, error) {
	f.mu.Lock()
	paths := slices.Clone(f.paths)
	f.mu.Unlock()

	out := []Favorite{}
	for _, p := range paths {
		path, info, err := Item(ctx, p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			out = append(out, Favorite{Name: filepath.Base(path), Path: path, Folder: path, Dir: true})
			continue
		}
		out = append(out, Favorite{Name: filepath.Base(path), Path: path, Folder: filepath.Dir(path), Size: uint64(info.Size())})
	}
	return out, nil
}

// Set ajoute (on) ou retire un fichier ou un dossier des favoris.
func (f *Favorites) Set(ctx context.Context, path string, on bool) error {
	if on {
		p, _, err := Item(ctx, path)
		if err != nil {
			return err
		}
		path = p
	} else {
		path = filepath.Clean(path)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	paths := slices.DeleteFunc(slices.Clone(f.paths), func(p string) bool { return pathKey(p) == pathKey(path) })
	if on {
		if len(paths) >= maxFavorites {
			return ErrTooMany
		}
		paths = append([]string{path}, paths...)
	}
	if err := f.save(paths); err != nil {
		return err
	}
	f.paths = paths
	return nil
}

func (f *Favorites) save(paths []string) error {
	b, err := json.MarshalIndent(stored{Paths: paths, Home: true}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}
