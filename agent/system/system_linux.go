package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// commands liste, par action, les commandes à essayer dans l'ordre. systemd/logind
// autorise ces actions à l'utilisateur connecté localement, sans sudo.
var commands = map[Action][][]string{
	Lock: {
		{"loginctl", "lock-session"},
		{"xdg-screensaver", "lock"},
		{"dbus-send", "--session", "--type=method_call", "--dest=org.freedesktop.ScreenSaver",
			"/org/freedesktop/ScreenSaver", "org.freedesktop.ScreenSaver.Lock"},
		{"gnome-screensaver-command", "--lock"},
	},
	Sleep:    {{"systemctl", "suspend"}},
	Restart:  {{"systemctl", "reboot"}},
	Shutdown: {{"systemctl", "poweroff"}},
}

func run(a Action) error {
	var errs []string
	for _, c := range commands[a] {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		out, err := exec.Command(c[0], c[1:]...).CombinedOutput()
		if err == nil {
			return nil
		}
		errs = append(errs, fmt.Sprintf("%s : %v %s", c[0], err, strings.TrimSpace(string(out))))
	}
	if len(errs) == 0 {
		return fmt.Errorf("aucune commande disponible pour %s", a)
	}
	return fmt.Errorf("%s", strings.Join(errs, " ; "))
}
