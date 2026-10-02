package autostart

import (
	"os"
	"strings"
	"testing"
)

func TestDesktopEntry(t *testing.T) {
	got := desktopEntry("/home/micka/mes outils/deckpad", []string{Flag, "-addr", ":9000"})
	if !strings.Contains(got, `Exec="/home/micka/mes outils/deckpad" -autostart -addr :9000`+"\n") {
		t.Errorf("Exec inattendu :\n%s", got)
	}
}

func TestEnableDisable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if enabled("") {
		t.Fatal("activé avant enable")
	}
	if err := enable("/usr/bin/deckpad", []string{Flag}); err != nil {
		t.Fatal(err)
	}
	if !enabled("") {
		t.Fatal("pas activé après enable")
	}
	p, _ := desktopPath()
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	if err := disable(); err != nil {
		t.Fatal(err)
	}
	if enabled("") {
		t.Fatal("encore activé après disable")
	}
	if err := disable(); err != nil {
		t.Fatalf("disable deux fois : %v", err)
	}
}
