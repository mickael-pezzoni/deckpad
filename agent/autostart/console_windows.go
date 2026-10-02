package autostart

import "golang.org/x/sys/windows"

// detachConsole ferme la fenêtre de console ouverte à la connexion.
func detachConsole() {
	windows.NewLazySystemDLL("kernel32.dll").NewProc("FreeConsole").Call()
}
