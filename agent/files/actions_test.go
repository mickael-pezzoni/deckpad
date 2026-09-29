package files

import (
	"context"
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
