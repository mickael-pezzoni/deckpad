package files

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Recent est un fichier ouvert (ou modifié) récemment sur le PC.
type Recent struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Folder string `json:"folder"` // dossier qui le contient : c'est lui que la tablette ouvre
	Size   uint64 `json:"size"`
	Used   int64  `json:"used"` // dernière utilisation, en millisecondes Unix
}

// recentItem : un candidat avant vérification (le fichier a pu être supprimé).
type recentItem struct {
	path string
	used time.Time
}

const maxRecents = 12

// Recents renvoie les fichiers récents, du plus récent au plus ancien : la liste
// tenue par le système (dossier « Récent » de Windows, recently-used.xbel sous
// Linux), complétée si besoin par les derniers fichiers modifiés dans les
// dossiers personnels (Bureau, Documents, Téléchargements…).
func Recents(ctx context.Context) ([]Recent, error) {
	drives, err := listDrives(ctx)
	if err != nil {
		return nil, err
	}
	items := systemRecents()
	recents := pickRecents(drives, items)
	if len(recents) < maxRecents {
		recents = pickRecents(drives, append(items, modifiedInUserFolders()...))
	}
	return recents, nil
}

// pickRecents garde les candidats qui existent encore, sur un disque visible, sans doublon.
func pickRecents(drives []Drive, items []recentItem) []Recent {
	sort.SliceStable(items, func(i, j int) bool { return items[i].used.After(items[j].used) })
	seen := map[string]bool{}
	recents := []Recent{}
	for _, it := range items {
		if len(recents) == maxRecents {
			break
		}
		path := filepath.Clean(it.path)
		key := pathKey(path)
		if seen[key] || !filepath.IsAbs(path) {
			continue
		}
		seen[key] = true
		if _, ok := driveOf(drives, path); !ok || hiddenPath(path) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || !isFile(info.Mode()) || isHidden(info.Name(), info) {
			continue
		}
		recents = append(recents, Recent{
			Name:   info.Name(),
			Path:   path,
			Folder: filepath.Dir(path),
			Size:   uint64(info.Size()),
			Used:   it.used.UnixMilli(),
		})
	}
	return recents
}

// modifiedInUserFolders parcourt les dossiers personnels et leurs sous-dossiers
// directs : de quoi trouver un document récent sans parcourir tout le disque.
func modifiedInUserFolders() []recentItem {
	const maxVisited = 5000 // borne le temps de réponse sur un dossier énorme
	visited := 0
	var items []recentItem
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if visited >= maxVisited {
				return
			}
			visited++
			info, err := e.Info()
			if err != nil || isHidden(e.Name(), info) {
				continue
			}
			path := filepath.Join(dir, e.Name())
			switch {
			case e.IsDir():
				if depth > 0 {
					walk(path, depth-1)
				}
			case isFile(info.Mode()):
				items = append(items, recentItem{path, info.ModTime()})
			}
		}
	}
	for _, dir := range userFolders() {
		walk(dir, 1)
	}
	return items
}
