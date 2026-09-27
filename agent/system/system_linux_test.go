package system

import "testing"

func TestEveryActionHasALinuxCommand(t *testing.T) {
	for _, a := range []Action{Lock, Sleep, Restart, Shutdown} {
		if len(commands[a]) == 0 {
			t.Errorf("aucune commande Linux pour %s", a)
		}
	}
}
