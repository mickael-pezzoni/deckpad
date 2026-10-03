//go:build windows || linux

// Package tray affiche l'icône de l'agent dans la zone de notification
// (Windows) ou via StatusNotifierItem (KDE, GNOME avec l'extension
// AppIndicator…) : démarrage avec Windows et arrêt de l'agent.
package tray

import (
	"log"
	"os"
	"runtime"

	"fyne.io/systray"

	"github.com/mickael-pezzoni/deckpad/agent/autostart"
	"github.com/mickael-pezzoni/deckpad/agent/icon"
)

// Run affiche l'icône et bloque jusqu'à « Quitter », qui arrête l'agent.
// Sans zone de notification (Linux sans bureau), il bloque simplement.
func Run(port string) {
	systray.Run(func() { ready(port) }, func() { os.Exit(0) })
}

func ready(port string) {
	if runtime.GOOS == "windows" {
		systray.SetIcon(icon.ICO)
	} else {
		systray.SetIcon(icon.PNG)
	}
	systray.SetTitle("deckpad")
	systray.SetTooltip("deckpad · port " + port)

	status := systray.AddMenuItem("deckpad · port "+port, "")
	status.Disable()
	systray.AddSeparator()

	// Démarrage auto : Windows seulement, l'entrée n'existe pas ailleurs.
	start := &systray.MenuItem{ClickedCh: make(chan struct{})}
	if st := autostart.Get(); st.Reason != autostart.ReasonOS {
		start = systray.AddMenuItemCheckbox(startLabel(st), "", st.Enabled)
		if !st.Supported {
			start.Disable()
		}
		systray.AddSeparator()
	}
	quit := systray.AddMenuItem("Quitter deckpad", "")

	for {
		select {
		case <-start.ClickedCh:
			st, err := autostart.Set(!start.Checked())
			if err != nil {
				log.Printf("démarrage auto : %v", err)
			}
			if st.Enabled {
				start.Check()
			} else {
				start.Uncheck()
			}
		case <-quit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func startLabel(st autostart.Status) string {
	label := "Lancer au démarrage de Windows"
	if st.Reason == autostart.ReasonDev {
		label += " (indisponible avec go run)"
	}
	return label
}
