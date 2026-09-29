package desktop

import (
	"cmp"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Sous X11, certains bureaux (KDE, GNOME) laissent la fenêtre ouverte derrière
// et affichent juste « prêt ». xdotool, déjà utilisé par Raccourcis, peut la
// passer devant : on photographie les fenêtres avant d'ouvrir, puis on active
// la nouvelle. Sous Wayland, seul le bureau décide : pas de premier plan forcé.

const (
	watchFor  = 8 * time.Second
	pollEvery = 250 * time.Millisecond
)

// x11Windows : fenêtres visibles, ou nil si on ne peut pas les activer.
func x11Windows() map[string]bool {
	if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland" || os.Getenv("DISPLAY") == "" {
		return nil
	}
	if _, err := exec.LookPath("xdotool"); err != nil {
		return nil
	}
	ids := map[string]bool{}
	for _, id := range listWindows() {
		ids[id] = true
	}
	return ids
}

func listWindows() []string {
	out, _ := exec.Command("xdotool", "search", "--onlyvisible", "--name", ".").Output()
	return strings.Fields(string(out))
}

// raise active la première nouvelle fenêtre qui accepte de l'être.
func raise(before map[string]bool) {
	for deadline := time.Now().Add(watchFor); time.Now().Before(deadline); time.Sleep(pollEvery) {
		var fresh []string
		for _, id := range listWindows() {
			if !before[id] {
				fresh = append(fresh, id)
			}
		}
		// Les identifiants croissent : la plus récente d'abord.
		slices.SortFunc(fresh, func(a, b string) int {
			x, _ := strconv.ParseUint(a, 10, 64)
			y, _ := strconv.ParseUint(b, 10, 64)
			return cmp.Compare(y, x)
		})
		for _, id := range fresh {
			if exec.Command("xdotool", "windowactivate", id).Run() == nil {
				return
			}
		}
	}
}

// openRaised lance la commande puis passe sa fenêtre au premier plan si possible.
func openRaised(name string, args ...string) error {
	return Launch(func() error { return startDetached(name, args...) })
}

// Launch lance un programme avec start, puis passe sa fenêtre au premier plan
// si possible.
func Launch(start func() error) error {
	before := x11Windows()
	if err := start(); err != nil {
		return err
	}
	if before != nil {
		go raise(before)
	}
	return nil
}
