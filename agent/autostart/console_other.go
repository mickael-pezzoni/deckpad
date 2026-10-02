//go:build !windows

package autostart

// detachConsole : rien à faire, le fichier .desktop demande déjà Terminal=false.
func detachConsole() {}
