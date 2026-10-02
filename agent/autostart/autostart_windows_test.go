package autostart

import "testing"

func TestCommandLine(t *testing.T) {
	got := commandLine(`C:\Program Files\deckpad\deckpad.exe`, []string{Flag, "-addr", ":9000"})
	want := `"C:\Program Files\deckpad\deckpad.exe" -autostart -addr :9000`
	if got != want {
		t.Errorf("commandLine = %s, attendu %s", got, want)
	}
}
