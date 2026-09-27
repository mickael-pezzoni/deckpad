// Package media pilote la lecture en cours sur le PC (page « Médias ») : titre,
// artiste, pochette, lecture/pause et morceau suivant/précédent. Il passe par les
// commandes multimédias du système, donc marche avec Spotify, YouTube dans le
// navigateur, VLC…
package media

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
)

type State struct {
	Active  bool   `json:"active"` // un lecteur est ouvert
	Playing bool   `json:"playing"`
	Title   string `json:"title,omitempty"`
	Artist  string `json:"artist,omitempty"`
	Album   string `json:"album,omitempty"`
	App     string `json:"app,omitempty"` // lecteur : Spotify, Chrome…
	// Cover change avec la pochette ("" : pas de pochette). La tablette l'ajoute
	// à l'adresse de l'image pour ne pas garder l'ancienne en cache.
	Cover string `json:"cover,omitempty"`
	// Unavailable explique pourquoi les médias ne sont pas pilotables.
	Unavailable string `json:"unavailable,omitempty"`

	coverRef string // où lire la pochette (propre à chaque système)
}

type Action string

const (
	PlayPause Action = "playpause"
	Next      Action = "next"
	Previous  Action = "previous"
)

var (
	ErrNoPlayer  = errors.New("aucun média en cours")
	ErrBadAction = errors.New("commande inconnue")
	ErrNoCover   = errors.New("pas de pochette")
	ErrRefused   = errors.New("le lecteur a refusé la commande")
)

// maxCover borne la taille d'une pochette lue ou téléchargée.
const maxCover = 8 << 20

var (
	mu       sync.Mutex
	coverRef string // pochette du dernier état lu
	cached   struct {
		ref  string
		data []byte
	}
)

// Collect lit ce qui est en cours de lecture.
func Collect(ctx context.Context) (State, error) {
	s, err := read(ctx)
	if err != nil {
		return State{Unavailable: err.Error()}, nil // la page affiche la raison
	}
	if s.coverRef != "" {
		sum := sha1.Sum([]byte(s.coverRef))
		s.Cover = hex.EncodeToString(sum[:6])
	}
	mu.Lock()
	coverRef = s.coverRef
	mu.Unlock()
	return s, nil
}

// Control envoie une commande au lecteur en cours.
func Control(ctx context.Context, a Action) error {
	switch a {
	case PlayPause, Next, Previous:
		return control(ctx, a)
	}
	return ErrBadAction
}

// Cover renvoie la pochette du morceau en cours (gardée en mémoire tant qu'il ne change pas).
func Cover(ctx context.Context) (data []byte, contentType string, err error) {
	mu.Lock()
	ref := coverRef
	if ref != "" && cached.ref == ref {
		data = cached.data
	}
	mu.Unlock()
	if ref == "" {
		return nil, "", ErrNoCover
	}
	if data == nil {
		if data, err = readCover(ctx, ref); err != nil {
			return nil, "", err
		}
		mu.Lock()
		cached.ref, cached.data = ref, data
		mu.Unlock()
	}
	contentType = http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", ErrNoCover
	}
	return data, contentType, nil
}

// appName met en forme un nom de programme : « spotify » → « Spotify ».
func appName(s string) string {
	s = strings.TrimSuffix(strings.TrimSuffix(s, ".exe"), ".EXE")
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// sourceApp donne un nom lisible au lecteur d'après son identifiant Windows
// (« Spotify.exe », « Chrome », « 308046B0AF4A39CB » pour Firefox…).
func sourceApp(id string) string {
	l := strings.ToLower(id)
	for _, k := range []struct{ key, name string }{
		{"spotify", "Spotify"},
		{"msedge", "Edge"},
		{"chrome", "Chrome"},
		{"308046b0af4a39cb", "Firefox"},
		{"firefox", "Firefox"},
		{"opera", "Opera"},
		{"brave", "Brave"},
		{"vivaldi", "Vivaldi"},
		{"vlc", "VLC"},
		{"zunemusic", "Lecteur multimédia"},
		{"zunevideo", "Films et TV"},
		{"deezer", "Deezer"},
		{"discord", "Discord"},
	} {
		if strings.Contains(l, k.key) {
			return k.name
		}
	}
	// Appli du Store : « Editeur.Appli_hash!App » ; programme : « C:\…\appli.exe ».
	name := id
	if i := strings.IndexByte(name, '!'); i >= 0 {
		name = name[:i]
	}
	if i := strings.IndexByte(name, '_'); i >= 0 {
		name = name[:i]
	}
	if i := strings.LastIndexAny(name, `\/`); i >= 0 {
		name = name[i+1:]
	}
	if i := strings.LastIndexByte(name, '.'); i >= 0 && !strings.EqualFold(name[i:], ".exe") {
		name = name[i+1:]
	}
	return appName(name)
}
