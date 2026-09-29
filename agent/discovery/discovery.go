// Package discovery annonce l'agent sur le réseau local en mDNS, pour que le
// serveur central deckpad le trouve sans rien configurer.
//
// Le type de service « _deckpad._tcp » et les clés TXT (id, v, https) sont
// partagés avec hub/discovery : les changer des deux côtés à la fois.
package discovery

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/libp2p/zeroconf/v2"
)

const (
	Service = "_deckpad._tcp"
	Domain  = "local."
	// Version du protocole annoncé, incrémentée si le serveur doit s'adapter.
	Version = "1"
)

// Announcer garde l'annonce active tant qu'elle n'est pas arrêtée.
type Announcer struct{ srv *zeroconf.Server }

// Announce publie l'agent : name est le nom affiché (celui du PC), port le port
// HTTP, httpsPort le port HTTPS (vide si désactivé), id un identifiant stable.
func Announce(name, id string, port int, httpsPort string) (*Announcer, error) {
	srv, err := zeroconf.Register(name, Service, Domain, port, Text(id, httpsPort), nil)
	if err != nil {
		return nil, err
	}
	return &Announcer{srv: srv}, nil
}

// Text construit les enregistrements TXT de l'annonce.
func Text(id, httpsPort string) []string {
	txt := []string{"id=" + id, "v=" + Version}
	if httpsPort != "" {
		txt = append(txt, "https="+httpsPort)
	}
	return txt
}

// Stop retire l'annonce (le serveur voit le PC disparaître tout de suite).
func (a *Announcer) Stop() {
	if a != nil {
		a.srv.Shutdown()
	}
}

// LoadID lit l'identifiant de cet agent dans path, ou le crée au premier
// lancement. Il ne change pas quand l'IP ou le nom du PC changent.
func LoadID(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		if id := strings.TrimSpace(string(b)); id != "" {
			return id, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("écriture de %s : %w", path, err)
	}
	return id, nil
}

// DefaultIDPath : %APPDATA%\deckpad\agent-id sous Windows, ~/.config/deckpad/agent-id sous Linux.
func DefaultIDPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "deckpad", "agent-id"), nil
}
