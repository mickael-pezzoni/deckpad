package autostart

import (
	"log"
	"os"
	"path/filepath"
)

// Background prépare un lancement sans fenêtre : le journal part dans
// deckpad/agent.log (dossier de config de l'utilisateur), puis la console est
// lâchée sous Windows.
func Background() {
	dir, err := os.UserConfigDir()
	if err == nil {
		dir = filepath.Join(dir, "deckpad")
		err = os.MkdirAll(dir, 0o700)
	}
	if err == nil {
		var f *os.File
		if f, err = os.Create(filepath.Join(dir, "agent.log")); err == nil {
			log.SetOutput(f)
		}
	}
	if err != nil {
		log.Printf("journal du démarrage auto : %v", err)
	}
	detachConsole()
}
