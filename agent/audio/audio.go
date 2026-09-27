// Package audio pilote le son du PC (page « Audio ») : volume général et par
// application, micro, choix de la sortie (casque, enceintes…).
package audio

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// Level est un volume de 0 à 100 et son état coupé.
type Level struct {
	Volume int  `json:"volume"`
	Muted  bool `json:"muted"`
}

// App regroupe les flux audio d'un même programme, comme le mélangeur de volume.
type App struct {
	ID   string `json:"id"`   // nom de l'exécutable en minuscules
	Name string `json:"name"` // nom à afficher
	Level
	exe string // chemin de l'exécutable, pour l'icône
}

// Device est une sortie audio (casque, enceintes, écran HDMI…).
type Device struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

type State struct {
	Master  *Level   `json:"master"` // nil : aucune sortie audio
	Mic     *Level   `json:"mic"`    // nil : aucun micro
	Outputs []Device `json:"outputs"`
	Apps    []App    `json:"apps"`
	// Unavailable explique pourquoi le son n'est pas pilotable (ex. pactl absent).
	Unavailable string `json:"unavailable,omitempty"`
}

var (
	ErrNotFound  = errors.New("application ou sortie audio introuvable")
	ErrBadTarget = errors.New("cible inconnue")
)

// Cibles d'un réglage de volume : "master", "mic" ou "app:<id>".
type target struct {
	kind string // master, mic ou app
	app  string
}

func parseTarget(s string) (target, error) {
	switch {
	case s == "master", s == "mic":
		return target{kind: s}, nil
	case strings.HasPrefix(s, "app:") && len(s) > 4:
		return target{kind: "app", app: s[4:]}, nil
	}
	return target{}, ErrBadTarget
}

var (
	exeMu sync.Mutex
	exes  = map[string]string{} // id d'appli → exécutable, vu à la dernière mesure
)

// Collect lit l'état du son. Les applications sont triées par nom.
func Collect(ctx context.Context) (State, error) {
	s, err := read(ctx)
	if err != nil {
		return State{Unavailable: err.Error()}, nil // la page affiche la raison
	}
	sort.Slice(s.Apps, func(i, j int) bool { return strings.ToLower(s.Apps[i].Name) < strings.ToLower(s.Apps[j].Name) })
	exeMu.Lock()
	for _, a := range s.Apps {
		exes[a.ID] = a.exe
	}
	exeMu.Unlock()
	if s.Outputs == nil {
		s.Outputs = []Device{}
	}
	if s.Apps == nil {
		s.Apps = []App{}
	}
	return s, nil
}

// AppExe renvoie l'exécutable d'une application audio (pour afficher son icône).
func AppExe(id string) (string, bool) {
	exeMu.Lock()
	defer exeMu.Unlock()
	p, ok := exes[id]
	return p, ok && p != ""
}

// SetVolume règle un volume de 0 à 100.
func SetVolume(ctx context.Context, t string, volume int) error {
	tg, err := parseTarget(t)
	if err != nil {
		return err
	}
	return setVolume(ctx, tg, min(max(volume, 0), 100))
}

// SetMute coupe ou rétablit le son (sortie, micro ou application).
func SetMute(ctx context.Context, t string, muted bool) error {
	tg, err := parseTarget(t)
	if err != nil {
		return err
	}
	return setMute(ctx, tg, muted)
}

// SetOutput choisit la sortie audio par défaut.
func SetOutput(ctx context.Context, id string) error {
	if id == "" {
		return ErrNotFound
	}
	return setOutput(ctx, id)
}

// appName met en forme un nom d'exécutable : « discord.exe » → « Discord ».
func appName(exe string) string {
	n := exe
	if strings.HasSuffix(strings.ToLower(n), ".exe") {
		n = n[:len(n)-4]
	}
	if n == "" {
		return exe
	}
	return strings.ToUpper(n[:1]) + n[1:]
}
