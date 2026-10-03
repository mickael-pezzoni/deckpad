// Package notify prévient la personne devant le PC qu'une action vient d'être
// faite depuis la tablette (fichier reçu, capture d'écran…), par une
// notification du système : toast Windows, ou serveur de notifications du
// bureau sous Linux (KDE, GNOME…). Sans bureau, la notification est ignorée.
package notify

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mickael-pezzoni/deckpad/agent/icon"
)

// Message : une notification, un titre court et une ligne de détail.
type Message struct {
	Title string
	Body  string
}

// show affiche la notification (selon le système), remplacé dans les tests.
var show = showSystem

var (
	startOnce sync.Once
	queue     = make(chan Message, 8)
)

// Send affiche la notification sans faire attendre l'appelant : la requête de
// la tablette répond tout de suite. Si plusieurs arrivent d'un coup, au-delà de
// quelques-unes en attente elles sont abandonnées plutôt que de s'empiler.
func Send(title, body string) {
	startOnce.Do(func() { go run() })
	select {
	case queue <- Message{Title: title, Body: body}:
	default:
	}
}

func run() {
	var failed bool
	for m := range queue {
		err := show(m)
		switch {
		case err != nil && !failed:
			// Une seule fois : sans bureau, chaque action le répéterait.
			log.Printf("notifications indisponibles : %v", err)
			failed = true
		case err == nil:
			failed = false
		}
	}
}

// Excerpt réduit un texte à une ligne d'au plus n caractères.
func Excerpt(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

var (
	iconOnce sync.Once
	iconFile string
)

// iconPath écrit l'icône de deckpad dans le cache, une fois : les deux systèmes
// veulent un fichier pour l'afficher dans la notification. "" en cas d'échec.
func iconPath() string {
	iconOnce.Do(func() {
		dir, err := os.UserCacheDir()
		if err != nil {
			return
		}
		dir = filepath.Join(dir, "deckpad")
		path := filepath.Join(dir, "icon.png")
		if old, err := os.ReadFile(path); err == nil && string(old) == string(icon.PNG) {
			iconFile = path
			return
		}
		if os.MkdirAll(dir, 0o755) != nil || os.WriteFile(path, icon.PNG, 0o644) != nil {
			return
		}
		iconFile = path
	})
	return iconFile
}
