//go:build !windows && !linux

package desktop

import "path/filepath"

// Open ouvre un dossier, un fichier ou une adresse avec l'appli associée.
func Open(target string) error { return startDetached("open", target) }

// Reveal ouvre le dossier du fichier.
func Reveal(path string) error { return Open(filepath.Dir(path)) }
