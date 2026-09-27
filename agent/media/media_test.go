package media

import "testing"

func TestSourceApp(t *testing.T) {
	for id, want := range map[string]string{
		"Spotify.exe": "Spotify",
		"SpotifyAB.SpotifyMusic_zpdnekdrzrea0!Spotify": "Spotify",
		"Chrome":           "Chrome",
		"MSEdge":           "Edge",
		"308046B0AF4A39CB": "Firefox",
		`C:\Program Files\foobar2000\foobar2000.exe`: "Foobar2000",
		"Contoso.Player_abc123!App":                  "Player",
		"mpv.exe":                                    "Mpv",
	} {
		if got := sourceApp(id); got != want {
			t.Errorf("sourceApp(%q) = %q, attendu %q", id, got, want)
		}
	}
}
