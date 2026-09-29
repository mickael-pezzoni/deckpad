package files

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// systemRecents lit le dossier « Récent » de l'utilisateur : un raccourci par
// fichier ouvert, daté du dernier accès (c'est la liste de l'Explorateur).
func systemRecents() []recentItem {
	dir, err := windows.KnownFolderPath(windows.FOLDERID_Recent, 0)
	if err != nil {
		return nil
	}
	entries, _ := os.ReadDir(dir)
	var items []recentItem
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".lnk") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		if target := lnkTarget(data); target != "" {
			items = append(items, recentItem{target, info.ModTime()})
		}
	}
	return items
}

// userFolders : Bureau, Documents, Téléchargements… là où Windows les place
// vraiment (ils peuvent avoir été déplacés, par OneDrive par exemple).
func userFolders() []string {
	var dirs []string
	for _, id := range []*windows.KNOWNFOLDERID{
		windows.FOLDERID_Desktop,
		windows.FOLDERID_Documents,
		windows.FOLDERID_Downloads,
		windows.FOLDERID_Pictures,
		windows.FOLDERID_Videos,
		windows.FOLDERID_Music,
	} {
		if dir, err := windows.KnownFolderPath(id, 0); err == nil {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}
