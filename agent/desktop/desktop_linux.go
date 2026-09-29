package desktop

import (
	"net/url"
	"path/filepath"

	"github.com/godbus/dbus/v5"
)

// Open ouvre un dossier, un fichier ou une adresse avec l'appli associée.
func Open(target string) error { return startDetached("xdg-open", target) }

// Reveal montre le fichier dans le gestionnaire de fichiers (Nautilus, Dolphin,
// Nemo… via l'interface FileManager1). À défaut, ouvre simplement son dossier.
func Reveal(path string) error {
	if showItem(path) == nil {
		return nil
	}
	return Open(filepath.Dir(path))
}

func showItem(path string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	uri := (&url.URL{Scheme: "file", Path: path}).String()
	obj := conn.Object("org.freedesktop.FileManager1", "/org/freedesktop/FileManager1")
	return obj.Call("org.freedesktop.FileManager1.ShowItems", 0, []string{uri}, "").Err
}
