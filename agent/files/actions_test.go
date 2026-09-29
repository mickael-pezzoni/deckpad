package files

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// onlyDrive fait comme si dir était le seul disque du PC.
func onlyDrive(t *testing.T, dir string) {
	t.Helper()
	old := fileDrives
	fileDrives = func(context.Context) ([]Drive, error) { return []Drive{{Path: dir}}, nil }
	t.Cleanup(func() { fileDrives = old })
}

func TestFile(t *testing.T) {
	dir := t.TempDir()
	onlyDrive(t, filepath.Join(dir, "disque"))
	file := filepath.Join(dir, "disque", "a.txt")
	os.MkdirAll(filepath.Dir(file), 0o700)
	os.WriteFile(file, []byte("abc"), 0o600)
	outside := filepath.Join(dir, "b.txt")
	os.WriteFile(outside, []byte("x"), 0o600)

	ctx := context.Background()
	if p, info, err := File(ctx, file); err != nil || p != file || info.Size() != 3 {
		t.Fatalf("File = %q, %v", p, err)
	}
	cases := map[string]error{
		outside:                             ErrOutside,
		"relatif.txt":                       ErrOutside,
		"":                                  ErrOutside,
		filepath.Join(dir, "disque"):        ErrNotFile,
		filepath.Join(dir, "disque", "zzz"): ErrNotFound,
	}
	for path, want := range cases {
		if _, _, err := File(ctx, path); !errors.Is(err, want) {
			t.Errorf("File(%q) = %v, attendu %v", path, err, want)
		}
	}
}

func TestFavorites(t *testing.T) {
	dir := t.TempDir()
	onlyDrive(t, dir)
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("a"), 0o600)
	os.WriteFile(b, []byte("bb"), 0o600)
	store := filepath.Join(dir, "conf", "favorites.json")
	old := homeDir
	homeDir = func() (string, error) { return "", errors.New("pas de dossier utilisateur") }
	t.Cleanup(func() { homeDir = old })

	ctx := context.Background()
	favs, err := OpenFavorites(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{a, b, a} { // a deux fois : pas de doublon, il remonte
		if err := favs.Set(ctx, p, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := favs.Set(ctx, filepath.Join(dir, "absent"), true); !errors.Is(err, ErrNotFound) {
		t.Errorf("favori absent : %v", err)
	}

	// Relu depuis le fichier.
	favs, err = OpenFavorites(store)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := favs.List(ctx)
	if len(list) != 2 || list[0].Path != a || list[1].Path != b || list[1].Size != 2 || list[1].Folder != dir || list[1].Name != "b.txt" {
		t.Fatalf("favoris = %+v", list)
	}

	// Un fichier supprimé disparaît de la liste ; on peut toujours le retirer.
	os.Remove(b)
	if list, _ := favs.List(ctx); len(list) != 1 {
		t.Errorf("favoris = %+v", list)
	}
	if err := favs.Set(ctx, b, false); err != nil {
		t.Fatal(err)
	}
	if err := favs.Set(ctx, a, false); err != nil {
		t.Fatal(err)
	}
	if list, _ := favs.List(ctx); len(list) != 0 {
		t.Errorf("favoris = %+v", list)
	}
}

func TestFavoriteFoldersAndHome(t *testing.T) {
	dir := t.TempDir()
	onlyDrive(t, dir)
	home := filepath.Join(dir, "home")
	games := filepath.Join(dir, "jeux")
	os.MkdirAll(home, 0o700)
	os.MkdirAll(games, 0o700)
	old := homeDir
	homeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { homeDir = old })
	store := filepath.Join(dir, "conf", "favorites.json")

	// Première ouverture : le dossier utilisateur est en favori.
	ctx := context.Background()
	favs, err := OpenFavorites(store)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := favs.List(ctx)
	if len(list) != 1 || list[0].Path != home || !list[0].Dir || list[0].Folder != home {
		t.Fatalf("favoris = %+v", list)
	}

	// Un dossier s'ajoute comme un fichier.
	if err := favs.Set(ctx, games, true); err != nil {
		t.Fatal(err)
	}
	// Retiré, le dossier utilisateur ne revient pas.
	if err := favs.Set(ctx, home, false); err != nil {
		t.Fatal(err)
	}
	favs, err = OpenFavorites(store)
	if err != nil {
		t.Fatal(err)
	}
	list, _ = favs.List(ctx)
	if len(list) != 1 || list[0].Path != games || !list[0].Dir || list[0].Name != "jeux" {
		t.Fatalf("favoris = %+v", list)
	}
}

func TestFavoritesOldFormat(t *testing.T) {
	dir := t.TempDir()
	onlyDrive(t, dir)
	home := filepath.Join(dir, "home")
	a := filepath.Join(dir, "a.txt")
	os.MkdirAll(home, 0o700)
	os.WriteFile(a, []byte("a"), 0o600)
	old := homeDir
	homeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { homeDir = old })
	store := filepath.Join(dir, "favorites.json")
	b, _ := json.Marshal([]string{a})
	os.WriteFile(store, b, 0o600)

	// Favoris d'avant : gardés, le dossier utilisateur vient après.
	favs, err := OpenFavorites(store)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := favs.List(context.Background())
	if len(list) != 2 || list[0].Path != a || list[1].Path != home {
		t.Fatalf("favoris = %+v", list)
	}
}
