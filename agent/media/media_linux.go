package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// Sous Linux, les lecteurs (Spotify, Firefox, Chromium, VLC, mpv…) publient leur
// lecture sur le bus de session D-Bus avec l'interface standard MPRIS. On lui
// parle directement : rien à installer.

const (
	mprisPrefix = "org.mpris.MediaPlayer2."
	mprisRoot   = "org.mpris.MediaPlayer2"
	mprisPlayer = "org.mpris.MediaPlayer2.Player"
	mprisPath   = dbus.ObjectPath("/org/mpris/MediaPlayer2")
)

var (
	busMu   sync.Mutex
	bus     *dbus.Conn
	current string // lecteur affiché (et piloté) en dernier
)

func session() (*dbus.Conn, error) {
	busMu.Lock()
	defer busMu.Unlock()
	if bus != nil && bus.Connected() {
		return bus, nil
	}
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("bus de session D-Bus inaccessible (lancez l'agent depuis votre session) : %w", err)
	}
	bus = c
	return c, nil
}

type player struct {
	name     string // nom sur le bus
	identity string
	status   string // Playing, Paused ou Stopped
	meta     map[string]dbus.Variant
}

func read(ctx context.Context) (State, error) {
	conn, err := session()
	if err != nil {
		return State{}, err
	}
	names, err := playerNames(ctx, conn)
	if err != nil {
		return State{}, err
	}
	var players []player
	for _, n := range names {
		if p, err := readPlayer(ctx, conn, n); err == nil {
			players = append(players, p)
		}
	}
	p := pick(players)
	if p == nil {
		return State{}, nil
	}
	return p.state(), nil
}

func playerNames(ctx context.Context, conn *dbus.Conn) ([]string, error) {
	var names []string
	if err := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return nil, fmt.Errorf("D-Bus : %w", err)
	}
	var out []string
	for _, n := range names {
		// playerctld recopie le lecteur actif : on l'aurait en double.
		if strings.HasPrefix(n, mprisPrefix) && n != mprisPrefix+"playerctld" {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out, nil
}

func readPlayer(ctx context.Context, conn *dbus.Conn, name string) (player, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	obj := conn.Object(name, mprisPath)
	var props map[string]dbus.Variant
	if err := obj.CallWithContext(ctx, "org.freedesktop.DBus.Properties.GetAll", 0, mprisPlayer).Store(&props); err != nil {
		return player{}, err
	}
	p := player{name: name}
	p.status, _ = props["PlaybackStatus"].Value().(string)
	p.meta, _ = props["Metadata"].Value().(map[string]dbus.Variant)
	var id dbus.Variant
	if err := obj.CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, mprisRoot, "Identity").Store(&id); err == nil {
		p.identity, _ = id.Value().(string)
	}
	if p.identity == "" {
		p.identity = appName(strings.SplitN(strings.TrimPrefix(name, mprisPrefix), ".", 2)[0])
	}
	return p, nil
}

// pick choisit le lecteur à afficher : celui qui joue, sinon celui déjà affiché,
// sinon le premier. Ainsi une vidéo en pause reste à l'écran.
func pick(players []player) *player {
	busMu.Lock()
	defer busMu.Unlock()
	var chosen *player
	for i := range players {
		p := &players[i]
		switch {
		case p.status == "Playing" && (chosen == nil || chosen.status != "Playing" || p.name == current):
			chosen = p
		case chosen == nil, chosen.status != "Playing" && p.name == current:
			chosen = p
		}
	}
	if chosen != nil {
		current = chosen.name
	} else {
		current = ""
	}
	return chosen
}

func (p *player) state() State {
	s := State{
		Active:   true,
		Playing:  p.status == "Playing",
		App:      p.identity,
		Title:    metaString(p.meta, "xesam:title"),
		Album:    metaString(p.meta, "xesam:album"),
		coverRef: metaString(p.meta, "mpris:artUrl"),
	}
	if artists, ok := p.meta["xesam:artist"].Value().([]string); ok {
		s.Artist = strings.Join(artists, ", ")
	} else {
		s.Artist = metaString(p.meta, "xesam:artist")
	}
	return s
}

func metaString(meta map[string]dbus.Variant, key string) string {
	v, ok := meta[key]
	if !ok {
		return ""
	}
	s, _ := v.Value().(string)
	return strings.TrimSpace(s)
}

func control(ctx context.Context, a Action) error {
	conn, err := session()
	if err != nil {
		return err
	}
	busMu.Lock()
	name := current
	busMu.Unlock()
	if name == "" {
		return ErrNoPlayer
	}
	method := map[Action]string{PlayPause: "PlayPause", Next: "Next", Previous: "Previous"}[a]
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := conn.Object(name, mprisPath).CallWithContext(ctx, mprisPlayer+"."+method, 0).Err; err != nil {
		var de dbus.Error
		if errors.As(err, &de) && de.Name == "org.freedesktop.DBus.Error.ServiceUnknown" {
			return ErrNoPlayer
		}
		return fmt.Errorf("%w : %v", ErrRefused, err)
	}
	return nil
}

// readCover lit la pochette annoncée par le lecteur : un fichier local
// (Firefox, Chromium, VLC) ou une adresse web (Spotify).
func readCover(ctx context.Context, ref string) ([]byte, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return nil, ErrNoCover
	}
	var r io.ReadCloser
	switch u.Scheme {
	case "file":
		if r, err = os.Open(u.Path); err != nil {
			return nil, ErrNoCover
		}
	case "http", "https":
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
		if err != nil {
			return nil, ErrNoCover
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, ErrNoCover
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, ErrNoCover
		}
		r = resp.Body
	default:
		return nil, ErrNoCover
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, maxCover))
	if err != nil || len(data) == 0 {
		return nil, ErrNoCover
	}
	return data, nil
}
