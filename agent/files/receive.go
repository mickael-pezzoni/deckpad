package files

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Réception d'un fichier envoyé depuis un autre PC (le hub fait le relais) :
// il arrive dans Téléchargements, renommé « nom (1).ext » si le nom est pris.

var ErrBadName = errors.New("nom de fichier invalide")

// receiveDir : le dossier de destination, remplacé dans les tests.
var receiveDir = downloadsDir

// Receive enregistre src sous name dans Téléchargements et renvoie le chemin final.
// Pendant la copie, le fichier s'appelle « .nom.deckpad » : un envoi interrompu
// ne laisse pas un fichier tronqué sous le vrai nom.
func Receive(name string, src io.Reader) (string, error) {
	name = safeName(name)
	if name == "" {
		return "", ErrBadName
	}
	dir, err := receiveDir()
	if err != nil {
		return "", err
	}
	// On réserve d'abord le nom final (création exclusive) pour ne jamais écraser.
	final, err := reserve(dir, name)
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*.deckpad")
	if err != nil {
		os.Remove(final)
		return "", err
	}
	// CreateTemp crée le fichier en 0600 : on lui donne les droits d'un fichier
	// ordinaire (sans effet utile sous Windows, d'où l'erreur ignorée).
	_ = tmp.Chmod(0o644)
	_, err = io.Copy(tmp, src)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), final)
	}
	if err != nil {
		os.Remove(tmp.Name())
		os.Remove(final)
		return "", err
	}
	return final, nil
}

// reserve crée un fichier vide au premier nom libre : « a.txt », « a (1).txt »…
func reserve(dir, name string) (string, error) {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	if stem == "" { // « .bashrc » : pas d'extension à préserver
		stem, ext = name, ""
	}
	for i := 0; i < 1000; i++ {
		candidate := name
		if i > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		path := filepath.Join(dir, candidate)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if errors.Is(err, fs.ErrPermission) {
			return "", ErrForbidden
		}
		if err != nil {
			return "", err
		}
		f.Close()
		return path, nil
	}
	return "", fs.ErrExist
}

// safeName garde le dernier élément du nom et remplace les caractères interdits
// sous Windows (un fichier Linux peut contenir « : » ou « ? »). Vide si inutilisable.
func safeName(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"|?*`, r) {
			return '_'
		}
		return r
	}, name)
	// Windows n'accepte pas un nom qui finit par un point ou une espace.
	name = strings.TrimRight(name, ". ")
	if name == "" || len(name) > 255 {
		return ""
	}
	return name
}
