package shortcuts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Sous Linux, aucun outil n'est commun à tous les bureaux : on prend le premier
// présent parmi ceux qui savent copier une capture de tout l'écran.
type captureTool struct {
	needs   []string
	wayland bool // ne marche que sous Wayland (sinon : que sous X11), sauf both
	both    bool
	cmd     string
}

var captureTools = []captureTool{
	{needs: []string{"gnome-screenshot"}, both: true, cmd: "gnome-screenshot -c"},
	{needs: []string{"spectacle"}, both: true, cmd: "spectacle -b -n -f -c"},
	{needs: []string{"grim", "wl-copy"}, wayland: true, cmd: "grim - | wl-copy -t image/png"},
	{needs: []string{"maim", "xclip"}, cmd: "maim | xclip -selection clipboard -t image/png"},
	{needs: []string{"import", "xclip"}, cmd: "import -window root png:- | xclip -selection clipboard -t image/png"},
}

func findCaptureTool() (captureTool, bool) {
	way := wayland()
	for _, t := range captureTools {
		if !t.both && t.wayland != way {
			continue
		}
		found := true
		for _, n := range t.needs {
			if _, err := exec.LookPath(n); err != nil {
				found = false
				break
			}
		}
		if found {
			return t, true
		}
	}
	return captureTool{}, false
}

func captureAvailable() Availability {
	if !wayland() && os.Getenv("DISPLAY") == "" {
		return Availability{Reason: "Aucune session graphique"}
	}
	if _, ok := findCaptureTool(); !ok {
		if wayland() {
			return Availability{Reason: "Installe gnome-screenshot, spectacle ou grim + wl-clipboard pour les captures"}
		}
		return Availability{Reason: "Installe gnome-screenshot, spectacle ou maim + xclip pour les captures"}
	}
	return Availability{OK: true}
}

func capture() error {
	if a := captureAvailable(); !a.OK {
		return fmt.Errorf("%s", a.Reason)
	}
	t, _ := findCaptureTool()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", t.cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s : %v %s", t.needs[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}
