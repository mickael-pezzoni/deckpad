package autostart

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const supported = true

// desktopPath : ~/.config/autostart/deckpad.desktop (spec XDG, suivie par
// GNOME, KDE, Xfce, Cinnamon…).
func desktopPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "deckpad.desktop"), nil
}

func enabled(string) bool {
	p, err := desktopPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

func enable(exe string, args []string) error {
	p, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(desktopEntry(exe, args)), 0o644)
}

func disable() error {
	p, err := desktopPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func desktopEntry(exe string, args []string) string {
	parts := []string{quote(exe)}
	for _, a := range args {
		parts = append(parts, quote(a))
	}
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=deckpad\n" +
		"Comment=Agent deckpad\n" +
		"Exec=" + strings.Join(parts, " ") + "\n" +
		"Terminal=false\n" +
		"NoDisplay=true\n" +
		"X-GNOME-Autostart-enabled=true\n"
}

// quote suit la spec Desktop Entry : guillemets doubles, \ " ` $ échappés, % doublé.
func quote(s string) string {
	s = strings.ReplaceAll(s, "%", "%%")
	if s != "" && !strings.ContainsAny(s, " \t\n\"'\\><~|&;$*?#()`") {
		return s
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", `\$`)
	return `"` + r.Replace(s) + `"`
}
