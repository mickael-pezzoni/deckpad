package server

import (
	"bytes"
	"encoding/json"
	"html"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

// pcName renvoie le nom court du PC ("MON-PC", sans ".local" ni domaine), "" si inconnu.
// Lu à chaque requête : un PC renommé est pris en compte sans relancer l'agent.
func pcName() string {
	host, err := os.Hostname()
	if err != nil {
		return ""
	}
	host, _, _ = strings.Cut(strings.TrimSpace(host), ".")
	return host
}

// namedManifest sert le manifeste de la PWA avec le nom du PC : à l'installation, l'icône
// sur la tablette porte ce nom, pratique pour distinguer plusieurs PC.
func namedManifest(dist fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := fs.ReadFile(dist, "manifest.webmanifest")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if host := pcName(); host != "" {
			var m map[string]any
			if json.Unmarshal(raw, &m) == nil {
				m["name"] = "deckpad · " + host
				m["short_name"] = host
				if out, err := json.MarshalIndent(m, "", "  "); err == nil {
					raw = out
				}
			}
		}
		// Type absent de la table de Go (et du registre Windows) : on le donne nous-mêmes.
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(raw))
	}
}

// namedIndex sert la page avec le nom du PC en titre (iOS le reprend sous l'icône).
func namedIndex(dist fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if host := pcName(); host != "" {
			h := html.EscapeString(host)
			raw = bytes.Replace(raw, []byte(`name="apple-mobile-web-app-title" content="deckpad"`),
				[]byte(`name="apple-mobile-web-app-title" content="`+h+`"`), 1)
			raw = bytes.Replace(raw, []byte("<title>deckpad</title>"),
				[]byte("<title>deckpad · "+h+"</title>"), 1)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Toujours revalider : une nouvelle version de l'exe doit arriver sur la tablette.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(raw))
	}
}
