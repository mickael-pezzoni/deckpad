// Package clipboard lit et remplit le presse-papiers du PC (page « Presse-papiers ») :
// texte envoyé depuis la tablette, et texte ou image copiés sur le PC.
package clipboard

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"unicode/utf8"
)

type State struct {
	Text string `json:"text,omitempty"`
	// Truncated : le texte est trop long pour la tablette, seul le début est envoyé.
	Truncated bool `json:"truncated,omitempty"`
	// Image change avec l'image copiée ("" : pas d'image). La tablette l'ajoute à
	// l'adresse de l'image pour ne pas garder l'ancienne en cache.
	Image string `json:"image,omitempty"`
	// Unavailable explique pourquoi le presse-papiers n'est pas lisible.
	Unavailable string `json:"unavailable,omitempty"`
}

var (
	ErrNoImage = errors.New("pas d'image dans le presse-papiers")
	ErrEmpty   = errors.New("texte vide")
	ErrTooLong = errors.New("texte trop long")
)

const (
	// MaxText borne le texte envoyé par la tablette.
	MaxText = 256 << 10
	// maxShown borne le texte renvoyé à la tablette (rafraîchi chaque seconde).
	maxShown = 64 << 10
	// maxImage borne la taille d'une image lue.
	maxImage = 32 << 20
)

// content : ce que le système a trouvé dans le presse-papiers. image est un PNG.
type content struct {
	text  string
	image []byte
}

var (
	mu        sync.Mutex
	lastImage struct {
		id   string
		data []byte
	}
)

// Collect lit le presse-papiers du PC.
func Collect(ctx context.Context) (State, error) {
	c, err := read(ctx)
	if err != nil {
		return State{Unavailable: err.Error()}, nil // la page affiche la raison
	}
	s := State{}
	s.Text, s.Truncated = shorten(c.text, maxShown)
	if len(c.image) > 0 {
		sum := sha1.Sum(c.image)
		s.Image = hex.EncodeToString(sum[:6])
	}
	mu.Lock()
	lastImage.id, lastImage.data = s.Image, c.image
	mu.Unlock()
	return s, nil
}

// Image renvoie l'image (PNG) lue en dernier par Collect.
func Image() ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	if len(lastImage.data) == 0 {
		return nil, ErrNoImage
	}
	return lastImage.data, nil
}

// SetText copie text dans le presse-papiers du PC.
func SetText(ctx context.Context, text string) error {
	if text == "" {
		return ErrEmpty
	}
	if len(text) > MaxText || !utf8.ValidString(text) {
		return ErrTooLong
	}
	return write(ctx, text)
}

// shorten coupe s à max octets au plus, sans couper un caractère.
func shorten(s string, max int) (string, bool) {
	if len(s) <= max {
		return s, false
	}
	s = s[:max]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return strings.TrimRight(s, "\r"), true
}
