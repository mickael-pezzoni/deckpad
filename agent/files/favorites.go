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

// Favorite est un fichier mis en favori depuis la tablette.
type Favorite struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Folder string `json:"folder"` // dossier qui le contient : c'est lui que la tablette ouvre
	Size   uint64 `json:"size"`
}

const maxFavorites = 50

var ErrTooMany = fmt.Errorf("%d favoris au maximum", maxFavorites)

// Favorites garde la liste des favoris dans favorites.json, à côté de devices.json.
type Favorites struct {
	path string

	mu    sync.Mutex
	paths []string // du plus récent au plus ancien
}

func DefaultFavoritesPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deckpad", "favorites.json"), nil
}

// OpenFavorites charge les favoris (fichier absent : aucun favori).
func OpenFavorites(path string) (*Favorites, error) {
	f := &Favorites{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &f.paths); err != nil {
		return nil, fmt.Errorf("%s illisible : %w", path, err)
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
		path, info, err := File(ctx, p)
		if err != nil {
			continue
		}
		out = append(out, Favorite{Name: filepath.Base(path), Path: path, Folder: filepath.Dir(path), Size: uint64(info.Size())})
	}
	return out, nil
}

// Set ajoute (on) ou retire un fichier des favoris.
func (f *Favorites) Set(ctx context.Context, path string, on bool) error {
	if on {
		p, _, err := File(ctx, path)
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
	b, err := json.MarshalIndent(paths, "", "  ")
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
