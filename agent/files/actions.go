package files

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mickael-pezzoni/deckpad/agent/desktop"
)

// Actions du menu d'un fichier sur la tablette : l'ouvrir sur le PC, le montrer
// dans l'Explorateur, le télécharger (voir le serveur) ou le mettre en favori.

var ErrNotFile = errors.New("ce n'est pas un fichier")

// fileDrives : les disques où chercher, remplacés dans les tests.
var fileDrives = listDrives

// File vérifie que path est un fichier visible sur l'un des disques et renvoie
// son chemin nettoyé et ses informations.
func File(ctx context.Context, path string) (string, fs.FileInfo, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", nil, ErrOutside
	}
	drives, err := fileDrives(ctx)
	if err != nil {
		return "", nil, err
	}
	path = filepath.Clean(path)
	if _, ok := driveOf(drives, path); !ok {
		return "", nil, ErrOutside
	}
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "", nil, ErrForbidden
	case errors.Is(err, fs.ErrNotExist):
		return "", nil, ErrNotFound
	case err != nil:
		return "", nil, err
	case info.IsDir() || !isFile(info.Mode()):
		return "", nil, ErrNotFile
	}
	return path, info, nil
}

// OpenOnPC ouvre le fichier sur le PC avec le programme associé.
func OpenOnPC(ctx context.Context, path string) error {
	path, _, err := File(ctx, path)
	if err != nil {
		return err
	}
	return desktop.Open(path)
}

// Reveal montre le fichier dans l'Explorateur (ou le gestionnaire de fichiers sous Linux).
func Reveal(ctx context.Context, path string) error {
	path, _, err := File(ctx, path)
	if err != nil {
		return err
	}
	return desktop.Reveal(path)
}
