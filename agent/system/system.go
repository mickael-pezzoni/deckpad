// Package system exécute les actions de la page « Système » : verrouiller,
// mettre en veille, redémarrer, éteindre.
package system

import (
	"errors"
	"log"
	"time"
)

type Action string

const (
	Lock     Action = "lock"
	Sleep    Action = "sleep"
	Restart  Action = "restart"
	Shutdown Action = "shutdown"
)

var ErrUnknown = errors.New("action inconnue")

// delay laisse le temps à la réponse HTTP d'arriver sur la tablette avant que le
// PC ne se verrouille, ne s'endorme ou ne s'éteigne.
const delay = 500 * time.Millisecond

// Run valide l'action puis la lance en arrière-plan.
func Run(a Action) error {
	switch a {
	case Lock, Sleep, Restart, Shutdown:
	default:
		return ErrUnknown
	}
	go func() {
		time.Sleep(delay)
		if err := run(a); err != nil {
			log.Printf("action %s : %v", a, err)
		}
	}()
	return nil
}
