package shortcuts

import "strconv"

// Modifiers dans l'ordre où ils sont appuyés. « win » est la touche Windows
// (Super sous Linux).
var Modifiers = []string{"ctrl", "alt", "shift", "win"}

// KeyNames : touches principales acceptées (une par raccourci).
var KeyNames = func() []string {
	var k []string
	for c := 'a'; c <= 'z'; c++ {
		k = append(k, string(c))
	}
	for c := '0'; c <= '9'; c++ {
		k = append(k, string(c))
	}
	for i := 1; i <= 12; i++ {
		k = append(k, "f"+strconv.Itoa(i))
	}
	return append(k,
		"esc", "enter", "tab", "space", "backspace", "delete", "insert",
		"printscreen", "up", "down", "left", "right", "home", "end", "pageup", "pagedown",
	)
}()

// Availability indique si le PC sait faire une action, et sinon pourquoi.
type Availability struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

// KeysAvailable indique si ce PC peut recevoir des combinaisons de touches.
func KeysAvailable() Availability { return keysAvailable() }

// CaptureAvailable indique si ce PC sait copier une capture d'écran.
func CaptureAvailable() Availability { return captureAvailable() }

// TypeAvailable indique si ce PC peut taper du texte envoyé par la tablette.
func TypeAvailable() Availability { return typeAvailable() }
