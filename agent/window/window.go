// Package window ouvre une petite fenêtre sur le PC (navigateur en mode application),
// utilisée pour afficher le code d'appairage.
package window

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Open affiche url dans une fenêtre sans barre d'adresse si Edge, Chrome ou Chromium
// est installé, sinon dans le navigateur par défaut.
func Open(url string) error {
	for _, browser := range browsers() {
		if err := start(browser, appArgs(url)...); err == nil {
			return nil
		}
	}
	name, args := fallback(url)
	if err := start(name, args...); err != nil {
		return fmt.Errorf("aucun navigateur n'a pu s'ouvrir : %w", err)
	}
	return nil
}

func appArgs(url string) []string {
	args := []string{
		"--app=" + url,
		"--window-size=460,420",
		"--no-first-run",
		"--no-default-browser-check",
	}
	// Profil dédié : la fenêtre s'ouvre à la bonne taille même si le navigateur
	// est déjà lancé, sans toucher aux onglets de l'utilisateur.
	if dir, err := os.UserCacheDir(); err == nil {
		args = append(args, "--user-data-dir="+filepath.Join(dir, "deckpad", "window"))
	}
	return args
}

func start(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // libère le processus quand la fenêtre se ferme
	return nil
}
