package files

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWithin(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "home")
	cases := map[string]bool{
		root:                         true,
		filepath.Join(root, "micka"): true,
		filepath.Join(root+"2", "x"): false,
		string(filepath.Separator):   false,
	}
	for path, want := range cases {
		if got := within(path, root); got != want {
			t.Errorf("within(%q, %q) = %v, attendu %v", path, root, got, want)
		}
	}
	if !within(filepath.Join(string(filepath.Separator), "etc"), string(filepath.Separator)) {
		t.Error("tout est sous la racine")
	}
}

func TestDriveOfPrefersDeepestMount(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("points de montage imbriqués : Linux")
	}
	drives := []Drive{{Path: "/"}, {Path: "/home"}}
	if root, _ := driveOf(drives, "/home/micka"); root != "/home" {
		t.Errorf("racine = %q", root)
	}
	if root, _ := driveOf(drives, "/etc"); root != "/" {
		t.Errorf("racine = %q", root)
	}
}

func TestSortEntries(t *testing.T) {
	e := []Entry{{Name: "b.txt"}, {Name: "Zeta", Dir: true}, {Name: "a.txt"}, {Name: "alpha", Dir: true}}
	sortEntries(e)
	want := []string{"alpha", "Zeta", "a.txt", "b.txt"}
	for i, name := range want {
		if e[i].Name != name {
			t.Fatalf("ordre = %v", e)
		}
	}
}

func TestIsFile(t *testing.T) {
	cases := map[fs.FileMode]bool{
		0:                 true,
		fs.ModeIrregular:  true, // fichier OneDrive sous Windows
		fs.ModeSocket:     false,
		fs.ModeNamedPipe:  false,
		fs.ModeDevice:     false,
		fs.ModeCharDevice: false,
	}
	for mode, want := range cases {
		if got := isFile(mode); got != want {
			t.Errorf("isFile(%v) = %v", mode, got)
		}
	}
}

func TestListRejectsOutside(t *testing.T) {
	drives, err := Drives(context.Background())
	if err != nil || len(drives) == 0 {
		t.Skip("aucun disque visible ici")
	}
	if _, err := List(context.Background(), "relatif"); !errors.Is(err, ErrOutside) {
		t.Errorf("chemin relatif : err = %v", err)
	}
}

func TestListDrive(t *testing.T) {
	drives, err := Drives(context.Background())
	if err != nil || len(drives) == 0 {
		t.Skip("aucun disque visible ici")
	}
	l, err := List(context.Background(), drives[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if l.Parent != "" {
		t.Errorf("racine du disque : parent = %q", l.Parent)
	}
	for _, e := range l.Entries {
		if e.Dir {
			sub, err := List(context.Background(), filepath.Join(l.Path, e.Name))
			if err == nil && sub.Parent != filepath.Clean(l.Path) {
				t.Errorf("parent = %q, attendu %q", sub.Parent, l.Path)
			}
			break
		}
	}
}
