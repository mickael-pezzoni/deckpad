package shortcuts

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Sous X11, xdotool envoie les touches. Sous Wayland, les applis n'ont pas le
// droit de simuler le clavier : il faut ydotool (et son service ydotoold).

var xdotoolNames = map[string]string{
	"ctrl": "ctrl", "alt": "alt", "shift": "shift", "win": "super",
	"esc": "Escape", "enter": "Return", "tab": "Tab", "space": "space",
	"backspace": "BackSpace", "delete": "Delete", "insert": "Insert", "printscreen": "Print",
	"up": "Up", "down": "Down", "left": "Left", "right": "Right",
	"home": "Home", "end": "End", "pageup": "Prior", "pagedown": "Next",
}

// Codes des touches du noyau Linux (input-event-codes.h), utilisés par ydotool.
var linuxKeyCodes = func() map[string]int {
	m := map[string]int{
		"ctrl": 29, "alt": 56, "shift": 42, "win": 125,
		"esc": 1, "enter": 28, "tab": 15, "space": 57, "backspace": 14,
		"delete": 111, "insert": 110, "printscreen": 99,
		"up": 103, "down": 108, "left": 105, "right": 106,
		"home": 102, "end": 107, "pageup": 104, "pagedown": 109,
		"f11": 87, "f12": 88, "0": 11,
	}
	for i, c := range "qwertyuiop" {
		m[string(c)] = 16 + i
	}
	for i, c := range "asdfghjkl" {
		m[string(c)] = 30 + i
	}
	for i, c := range "zxcvbnm" {
		m[string(c)] = 44 + i
	}
	for i := 1; i <= 9; i++ {
		m[strconv.Itoa(i)] = 1 + i
	}
	for i := 1; i <= 10; i++ {
		m["f"+strconv.Itoa(i)] = 58 + i
	}
	return m
}()

func wayland() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
}

func keysAvailable() Availability {
	switch {
	case wayland():
		if _, err := exec.LookPath("ydotool"); err != nil {
			return Availability{Reason: "Sous Wayland, installe ydotool pour envoyer des touches"}
		}
	case os.Getenv("DISPLAY") != "":
		if _, err := exec.LookPath("xdotool"); err != nil {
			return Availability{Reason: "Installe xdotool pour envoyer des touches"}
		}
	default:
		return Availability{Reason: "Aucune session graphique"}
	}
	return Availability{OK: true}
}

func sendKeys(keys []string) error {
	if s := keysAvailable(); !s.OK {
		return fmt.Errorf("%s", s.Reason)
	}
	var cmd *exec.Cmd
	if wayland() {
		cmd = exec.Command("ydotool", append([]string{"key"}, ydotoolArgs(keys)...)...)
	} else {
		cmd = exec.Command("xdotool", "key", "--clearmodifiers", xdotoolCombo(keys))
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s : %v %s", cmd.Args[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}

// xdotoolCombo : ["ctrl", "shift", "f5"] → "ctrl+shift+F5".
func xdotoolCombo(keys []string) string {
	names := make([]string, len(keys))
	for i, k := range keys {
		switch {
		case xdotoolNames[k] != "":
			names[i] = xdotoolNames[k]
		case len(k) > 1 && k[0] == 'f':
			names[i] = "F" + k[1:]
		default:
			names[i] = k
		}
	}
	return strings.Join(names, "+")
}

// ydotoolArgs : appuis dans l'ordre (code:1), relâchements dans l'ordre inverse (code:0).
func ydotoolArgs(keys []string) []string {
	var args []string
	for _, k := range keys {
		args = append(args, fmt.Sprintf("%d:1", linuxKeyCodes[k]))
	}
	for i := len(keys) - 1; i >= 0; i-- {
		args = append(args, fmt.Sprintf("%d:0", linuxKeyCodes[keys[i]]))
	}
	return args
}

func launch(command string) error { return startDetached("sh", "-c", command) }

// open ouvre un dossier, un fichier ou une adresse avec l'appli associée.
func open(target string) error { return startDetached("xdg-open", target) }

func defaults() []Shortcut {
	return []Shortcut{
		{ID: "capture", Label: "Capture d'écran", Icon: "camera", Color: "blue", Kind: KindCapture},
		{ID: "record", Label: "Enregistrer l'écran", Icon: "video", Color: "red", Kind: KindKeys, Keys: []string{"ctrl", "alt", "shift", "r"}},
		{ID: "taskmgr", Label: "Moniteur système", Icon: "gauge", Color: "green", Kind: KindLaunch,
			Command: "gnome-system-monitor || plasma-systemmonitor || ksysguard || xfce4-taskmanager"},
		{ID: "home", Label: "Dossier personnel", Icon: "folder", Color: "orange", Kind: KindOpen, Target: "~"},
		{ID: "desktop", Label: "Afficher le bureau", Icon: "monitor", Color: "purple", Kind: KindKeys, Keys: []string{"win", "d"}},
		{ID: "calc", Label: "Calculatrice", Icon: "calculator", Color: "teal", Kind: KindLaunch,
			Command: "gnome-calculator || kcalc || galculator || xcalc"},
	}
}

// Pour taper du texte sous Wayland, wtype comprend tous les caractères mais ne
// marche pas sous GNOME ; ydotool marche partout mais tape selon un clavier
// QWERTY (les accents et la disposition AZERTY peuvent donner d'autres lettres).
func typeAvailable() Availability {
	switch {
	case wayland():
		if _, err := exec.LookPath("wtype"); err == nil {
			return Availability{OK: true}
		}
		if _, err := exec.LookPath("ydotool"); err != nil {
			return Availability{Reason: "Sous Wayland, installe wtype ou ydotool pour taper du texte"}
		}
	case os.Getenv("DISPLAY") != "":
		if _, err := exec.LookPath("xdotool"); err != nil {
			return Availability{Reason: "Installe xdotool pour taper du texte"}
		}
	default:
		return Availability{Reason: "Aucune session graphique"}
	}
	return Availability{OK: true}
}

func typeText(text string) error {
	var cmds [][]string
	if wayland() {
		cmds = [][]string{{"wtype", "--", text}, {"ydotool", "type", "--", text}}
	} else {
		cmds = [][]string{{"xdotool", "type", "--clearmodifiers", "--delay", "8", "--", text}}
	}
	var err error
	for _, c := range cmds {
		if _, e := exec.LookPath(c[0]); e != nil {
			continue
		}
		out, e := exec.Command(c[0], c[1:]...).CombinedOutput()
		if e == nil {
			return nil
		}
		err = fmt.Errorf("%s : %v %s", c[0], e, strings.TrimSpace(string(out)))
	}
	return err
}
