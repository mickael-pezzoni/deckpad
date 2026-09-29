//go:build !windows

package files

import (
	"bufio"
	"encoding/xml"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// systemRecents lit recently-used.xbel, la liste partagée par les applis
// GNOME, KDE, les navigateurs… (spécification freedesktop).
func systemRecents() []recentItem {
	data, err := os.ReadFile(filepath.Join(dataHome(), "recently-used.xbel"))
	if err != nil {
		return nil
	}
	return parseXBEL(data)
}

func parseXBEL(data []byte) []recentItem {
	var doc struct {
		Bookmarks []struct {
			Href     string `xml:"href,attr"`
			Modified string `xml:"modified,attr"`
			Visited  string `xml:"visited,attr"`
		} `xml:"bookmark"`
	}
	if xml.Unmarshal(data, &doc) != nil {
		return nil
	}
	var items []recentItem
	for _, b := range doc.Bookmarks {
		u, err := url.Parse(b.Href)
		if err != nil || u.Scheme != "file" {
			continue
		}
		used := latest(b.Modified, b.Visited)
		items = append(items, recentItem{u.Path, used})
	}
	return items
}

func latest(dates ...string) time.Time {
	var t time.Time
	for _, d := range dates {
		if v, err := time.Parse(time.RFC3339, d); err == nil && v.After(t) {
			t = v
		}
	}
	return t
}

func dataHome() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}

// userFolders : les dossiers XDG (« Bureau », « Téléchargements »… selon la
// langue) lus dans user-dirs.dirs, ou leurs noms anglais par défaut.
func userFolders() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	dirs := map[string]string{
		"DESKTOP": "Desktop", "DOCUMENTS": "Documents", "DOWNLOAD": "Downloads",
		"PICTURES": "Pictures", "VIDEOS": "Videos", "MUSIC": "Music",
	}
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	}
	if f, err := os.Open(filepath.Join(config, "user-dirs.dirs")); err == nil {
		for key, dir := range parseUserDirs(bufio.NewScanner(f), home) {
			if _, ok := dirs[key]; ok {
				dirs[key] = dir
			}
		}
		f.Close()
	}
	var out []string
	for _, key := range []string{"DESKTOP", "DOCUMENTS", "DOWNLOAD", "PICTURES", "VIDEOS", "MUSIC"} {
		dir := dirs[key]
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(home, dir)
		}
		if dir != home { // un dossier XDG désactivé pointe sur le dossier personnel
			out = append(out, dir)
		}
	}
	return out
}

// parseUserDirs lit les lignes XDG_DOWNLOAD_DIR="$HOME/Téléchargements".
func parseUserDirs(s *bufio.Scanner, home string) map[string]string {
	dirs := map[string]string{}
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		key, value, ok := strings.Cut(line, "=")
		if !ok || !strings.HasPrefix(key, "XDG_") || !strings.HasSuffix(key, "_DIR") {
			continue
		}
		value = strings.ReplaceAll(strings.Trim(value, `"`), "$HOME", home)
		dirs[strings.TrimSuffix(strings.TrimPrefix(key, "XDG_"), "_DIR")] = value
	}
	return dirs
}
