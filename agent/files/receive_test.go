package files

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeName(t *testing.T) {
	for in, want := range map[string]string{
		"photo.jpg":              "photo.jpg",
		"../../etc/passwd":       "passwd",
		`C:\Users\a\b.txt`:       "b.txt",
		"notes: v2?.md":          "notes_ v2_.md",
		"..":                     "",
		"":                       "",
		"fin. ":                  "fin",
		strings.Repeat("a", 300): "",
	} {
		if got := safeName(in); got != want {
			t.Errorf("safeName(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestReceive(t *testing.T) {
	dir := t.TempDir()
	receiveDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { receiveDir = downloadsDir })

	for i, want := range []string{"rapport.pdf", "rapport (1).pdf", "rapport (2).pdf"} {
		path, err := Receive(context.Background(), "", "rapport.pdf", strings.NewReader("contenu"))
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(path) != want {
			t.Errorf("envoi %d : %s, attendu %s", i, filepath.Base(path), want)
		}
		if b, _ := os.ReadFile(path); string(b) != "contenu" {
			t.Errorf("contenu de %s : %q", want, b)
		}
	}
	if path, err := Receive(context.Background(), "", ".bashrc", strings.NewReader("")); err != nil || filepath.Base(path) != ".bashrc" {
		t.Errorf("fichier caché : %s, %v", path, err)
	}
	if _, err := Receive(context.Background(), "", "..", strings.NewReader("")); !errors.Is(err, ErrBadName) {
		t.Errorf("nom invalide accepté : %v", err)
	}

	// Vers un dossier choisi : il doit exister sur un disque et être un dossier.
	onlyDrive(t, dir)
	sub := filepath.Join(dir, "Photos")
	os.Mkdir(sub, 0o755)
	if path, err := Receive(context.Background(), sub, "a.jpg", strings.NewReader("x")); err != nil || filepath.Dir(path) != sub {
		t.Errorf("dossier choisi : %s, %v", path, err)
	}
	if _, err := Receive(context.Background(), filepath.Join(dir, "absent"), "a.jpg", strings.NewReader("x")); !errors.Is(err, ErrNotFound) {
		t.Errorf("dossier absent : %v", err)
	}
	if _, err := Receive(context.Background(), filepath.Join(sub, "a.jpg"), "b.jpg", strings.NewReader("x")); !errors.Is(err, ErrNotDir) {
		t.Errorf("fichier pris pour un dossier : %v", err)
	}

	// Envoi interrompu : ni fichier tronqué, ni nom réservé qui traîne.
	if _, err := Receive(context.Background(), "", "coupe.bin", io.MultiReader(strings.NewReader("début"), failing{})); err == nil {
		t.Error("erreur de lecture ignorée")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "coupe") || strings.HasSuffix(e.Name(), ".deckpad") {
			t.Errorf("reste d'un envoi interrompu : %s", e.Name())
		}
	}
}

type failing struct{}

func (failing) Read([]byte) (int, error) { return 0, errors.New("connexion coupée") }
