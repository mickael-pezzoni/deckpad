// Package files liste les disques du PC et le contenu de leurs dossiers
// (page « Fichiers »). Lecture seule : rien n'est ouvert, modifié ou supprimé.
package files

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Drive est un disque (ou une partition) tel que l'utilisateur le voit :
// « C: » sous Windows, un point de montage sous Linux.
type Drive struct {
	Path      string `json:"path"`      // racine à parcourir : « C:\ » ou « /home »
	Name      string `json:"name"`      // nom du volume, ou un nom lisible à défaut
	Total     uint64 `json:"total"`     // octets
	Used      uint64 `json:"used"`      // octets
	Removable bool   `json:"removable"` // clé ou disque USB
}

type Entry struct {
	Name string `json:"name"`
	Dir  bool   `json:"dir"`
	Size uint64 `json:"size"` // fichiers seulement
}

type Listing struct {
	Path      string  `json:"path"`
	Parent    string  `json:"parent"` // vide à la racine du disque
	Entries   []Entry `json:"entries"`
	Truncated int     `json:"truncated"` // éléments non envoyés (dossier énorme)
}

// maxEntries borne la réponse : au-delà, la tablette ne pourrait pas tout afficher.
const maxEntries = 500

var (
	ErrOutside   = errors.New("dossier hors des disques")
	ErrForbidden = errors.New("accès refusé")
	ErrNotFound  = errors.New("dossier introuvable")
)

// Drives renvoie les disques du PC, dans l'ordre (C:, D:… ou / puis le reste).
func Drives(ctx context.Context) ([]Drive, error) {
	drives, err := listDrives(ctx)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(drives, func(i, j int) bool {
		if drives[i].Removable != drives[j].Removable {
			return !drives[i].Removable // disques internes d'abord
		}
		return drives[i].Path < drives[j].Path
	})
	return drives, nil
}

// List renvoie les sous-dossiers puis les fichiers de path, sans les éléments cachés.
// path doit se trouver sur l'un des disques renvoyés par Drives.
func List(ctx context.Context, path string) (*Listing, error) {
	drives, err := listDrives(ctx)
	if err != nil {
		return nil, err
	}
	path = filepath.Clean(path)
	root, ok := driveOf(drives, path)
	if !ok {
		return nil, ErrOutside
	}

	dirEntries, err := os.ReadDir(path)
	switch {
	case errors.Is(err, fs.ErrPermission):
		return nil, ErrForbidden
	case errors.Is(err, fs.ErrNotExist):
		return nil, ErrNotFound
	case err != nil && len(dirEntries) == 0:
		return nil, err
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, d := range dirEntries {
		info, err := d.Info()
		if err != nil || isHidden(d.Name(), info) {
			continue
		}
		e := Entry{Name: d.Name(), Dir: d.IsDir()}
		if info.Mode()&fs.ModeSymlink != 0 {
			// Lien : on regarde ce qu'il vise pour savoir si on peut y entrer.
			target, err := os.Stat(filepath.Join(path, d.Name()))
			if err != nil {
				continue
			}
			info = target
			e.Dir = target.IsDir()
		}
		if !e.Dir {
			if !info.Mode().IsRegular() {
				continue // sockets, périphériques… rien d'utile à montrer
			}
			e.Size = uint64(info.Size())
		}
		entries = append(entries, e)
	}
	sortEntries(entries)

	l := &Listing{Path: path, Entries: entries}
	if len(entries) > maxEntries {
		l.Entries = entries[:maxEntries]
		l.Truncated = len(entries) - maxEntries
	}
	if !samePath(path, root) {
		l.Parent = filepath.Dir(path)
	}
	return l, nil
}

// sortEntries : dossiers d'abord, puis ordre alphabétique sans tenir compte de la casse.
func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

// driveOf renvoie la racine du disque qui contient path (la plus précise si
// des points de montage sont imbriqués, ex. / et /home).
func driveOf(drives []Drive, path string) (string, bool) {
	best := ""
	for _, d := range drives {
		root := filepath.Clean(d.Path)
		if within(path, root) && len(root) > len(best) {
			best = root
		}
	}
	return best, best != ""
}

// within indique si path est root ou se trouve dessous (chemins déjà nettoyés).
func within(path, root string) bool {
	if samePath(path, root) {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return len(path) > len(prefix) && samePath(path[:len(prefix)], prefix)
}
