package shortcuts

import (
	"slices"
	"testing"
)

func TestEveryKeyHasALinuxCode(t *testing.T) {
	for _, k := range append(slices.Clone(Modifiers), KeyNames...) {
		if _, ok := linuxKeyCodes[k]; !ok {
			t.Errorf("pas de code pour %q", k)
		}
	}
}

func TestXdotoolCombo(t *testing.T) {
	if got := xdotoolCombo([]string{"ctrl", "win", "f5"}); got != "ctrl+super+F5" {
		t.Errorf("obtenu %q", got)
	}
	if got := xdotoolCombo([]string{"shift", "pageup"}); got != "shift+Prior" {
		t.Errorf("obtenu %q", got)
	}
}

func TestYdotoolArgs(t *testing.T) {
	got := ydotoolArgs([]string{"ctrl", "c"})
	if want := []string{"29:1", "46:1", "46:0", "29:0"}; !slices.Equal(got, want) {
		t.Errorf("obtenu %v, attendu %v", got, want)
	}
}
