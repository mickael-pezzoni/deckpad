package audio

import "testing"

func TestParseTarget(t *testing.T) {
	for in, want := range map[string]target{
		"master":      {kind: "master"},
		"mic":         {kind: "mic"},
		"app:discord": {kind: "app", app: "discord"},
	} {
		if got, err := parseTarget(in); err != nil || got != want {
			t.Errorf("parseTarget(%q) = %+v, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "app:", "discord"} {
		if _, err := parseTarget(in); err == nil {
			t.Errorf("parseTarget(%q) : erreur attendue", in)
		}
	}
}

func TestAppName(t *testing.T) {
	for in, want := range map[string]string{"discord.exe": "Discord", "Spotify.EXE": "Spotify", "firefox": "Firefox"} {
		if got := appName(in); got != want {
			t.Errorf("appName(%q) = %q, attendu %q", in, got, want)
		}
	}
}
