package media

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestPick(t *testing.T) {
	ps := func(statuses ...string) []player {
		var out []player
		for i, s := range statuses {
			out = append(out, player{name: string(rune('a' + i)), status: s})
		}
		return out
	}
	cases := []struct {
		name    string
		current string
		players []player
		want    string
	}{
		{"aucun lecteur", "a", nil, ""},
		{"celui qui joue", "", ps("Paused", "Playing"), "b"},
		{"garde celui en pause déjà affiché", "b", ps("Stopped", "Paused"), "b"},
		{"celui qui joue passe devant", "a", ps("Paused", "Playing"), "b"},
		{"deux qui jouent : garde l'actuel", "c", ps("Playing", "Paused", "Playing"), "c"},
		{"lecteur fermé : le premier", "z", ps("Paused", "Stopped"), "a"},
	}
	for _, c := range cases {
		current = c.current
		got := ""
		if p := pick(c.players); p != nil {
			got = p.name
		}
		if got != c.want || current != c.want {
			t.Errorf("%s : %q (actuel %q), attendu %q", c.name, got, current, c.want)
		}
	}
}

func TestPlayerState(t *testing.T) {
	p := player{identity: "Spotify", status: "Playing", meta: map[string]dbus.Variant{
		"xesam:title":  dbus.MakeVariant("Get Lucky"),
		"xesam:artist": dbus.MakeVariant([]string{"Daft Punk", "Pharrell Williams"}),
		"xesam:album":  dbus.MakeVariant("Random Access Memories"),
		"mpris:artUrl": dbus.MakeVariant("https://i.scdn.co/image/abc"),
	}}
	s := p.state()
	if !s.Active || !s.Playing || s.Title != "Get Lucky" || s.Artist != "Daft Punk, Pharrell Williams" ||
		s.App != "Spotify" || s.coverRef != "https://i.scdn.co/image/abc" {
		t.Errorf("état inattendu : %+v", s)
	}
}
