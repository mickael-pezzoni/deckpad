// Package autostart lance l'agent à l'ouverture de session de l'utilisateur :
// clé Run du registre sous Windows, fichier .desktop de ~/.config/autostart sous Linux.
// Démarrer avec la session (et non comme service) garde l'accès au bureau :
// premier plan, touches, presse-papiers.
package autostart

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Flag est ajouté à la commande enregistrée : l'agent sait qu'il démarre seul
// (pas de console, journal dans un fichier).
const Flag = "-autostart"

// Raisons pour lesquelles le démarrage auto n'est pas proposé.
const (
	ReasonDev = "dev" // lancé par go run : l'exécutable est temporaire
	ReasonOS  = "os"  // système non pris en charge
)

type Status struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Reason    string `json:"reason,omitempty"`
}

var ErrUnsupported = errors.New("démarrage automatique indisponible")

// Get indique si l'agent démarre avec la session.
func Get() Status {
	if !supported {
		return Status{Reason: ReasonOS}
	}
	exe, err := executable()
	if err != nil {
		return Status{Reason: ReasonDev}
	}
	return Status{Supported: true, Enabled: enabled(exe)}
}

// Set active ou retire le démarrage avec la session, puis relit l'état.
func Set(on bool) (Status, error) {
	err := set(on)
	return Get(), err
}

func set(on bool) error {
	if !supported {
		return ErrUnsupported
	}
	if !on {
		return disable()
	}
	exe, err := executable()
	if err != nil {
		return ErrUnsupported
	}
	return enable(exe, args())
}

// executable renvoie le chemin de l'agent, sauf s'il vient de go run : ce
// binaire est effacé à l'arrêt, l'enregistrer ne servirait à rien.
// Variable pour les tests, lancés eux-mêmes depuis go-build.
var executable = func() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	if isTemporary(exe) {
		return "", ErrUnsupported
	}
	return exe, nil
}

func isTemporary(exe string) bool {
	return strings.Contains(strings.ReplaceAll(exe, `\`, "/"), "/go-build")
}

// args reprend les options de lancement actuelles (ex : -addr), plus Flag.
func args() []string {
	out := []string{Flag}
	for _, a := range os.Args[1:] {
		if a != Flag {
			out = append(out, a)
		}
	}
	return out
}
