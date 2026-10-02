package autostart

import "testing"

func TestIsTemporary(t *testing.T) {
	for exe, want := range map[string]bool{
		`C:\Users\micka\AppData\Local\Temp\go-build123\b001\exe\agent.exe`: true,
		"/tmp/go-build4567/b001/exe/agent":                                 true,
		`C:\Program Files\deckpad\deckpad.exe`:                             false,
		"/home/micka/bin/deckpad":                                          false,
	} {
		if got := isTemporary(exe); got != want {
			t.Errorf("isTemporary(%q) = %v, attendu %v", exe, got, want)
		}
	}
}
