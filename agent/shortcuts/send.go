package shortcuts

import (
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"
)

// Envois depuis la page « Presse-papiers » : ouvrir un lien ou taper un texte.

var (
	ErrBadURL   = errors.New("ce n'est pas une adresse web (http ou https)")
	ErrLongText = errors.New("texte trop long pour être tapé")
)

// maxTyped borne le texte tapé : au-delà, mieux vaut le copier puis le coller.
const maxTyped = 5000

// OpenURL ouvre une adresse web dans le navigateur par défaut du PC. Seules les
// adresses http et https sont acceptées : un chemin ou un programme serait lancé.
func OpenURL(raw string) error {
	u, err := webURL(raw)
	if err != nil {
		return err
	}
	return open(u)
}

// webURL vérifie une adresse web ; « exemple.fr/page » devient « https://exemple.fr/page ».
func webURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") && strings.Contains(raw, ".") && !strings.ContainsAny(raw, " \t\n") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", ErrBadURL
	}
	return u.String(), nil
}

// TypeText tape text dans la fenêtre active du PC, comme au clavier.
func TypeText(text string) error {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	if text == "" {
		return errors.New("texte vide")
	}
	if utf8.RuneCountInString(text) > maxTyped {
		return ErrLongText
	}
	if s := typeAvailable(); !s.OK {
		return errors.New(s.Reason)
	}
	return typeText(text)
}
