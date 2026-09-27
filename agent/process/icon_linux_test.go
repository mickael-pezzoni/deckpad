package process

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxIconLookup(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	write("applications/firefox.desktop", "[Desktop Entry]\nName=Firefox\nExec=env MOZ_X=1 /usr/lib/firefox/firefox %u\nIcon=firefox\n[Desktop Action new]\nIcon=autre\n")
	write("applications/steam.desktop", "[Desktop Entry]\nExec=\"/usr/games/steam\" %U\nIcon=steam\nStartupWMClass=Steam\n")
	write("icons/hicolor/48x48/apps/firefox.png", "png48")
	write("icons/hicolor/64x64/apps/firefox.png", "png64")
	write("icons/hicolor/scalable/apps/steam.svg", "<svg/>")

	dirs := []string{root}
	index := indexDesktopFiles(dirs)
	if index["firefox"] != "firefox" {
		t.Fatalf("firefox : %q (index %v)", index["firefox"], index)
	}
	if index["steam"] != "steam" {
		t.Errorf("steam : %q", index["steam"])
	}

	if got := resolveIcon(dirs, "firefox"); filepath.Base(filepath.Dir(filepath.Dir(got))) != "64x64" {
		t.Errorf("la taille 64x64 doit être préférée : %s", got)
	}
	ic, err := readIcon(resolveIcon(dirs, "steam"))
	if err != nil || ic.ContentType != "image/svg+xml" {
		t.Errorf("icône SVG attendue : %+v, %v", ic, err)
	}
	write("pixmaps/vieux.xpm", "xpm")
	write("pixmaps/vieux.png", "png")
	if got := resolveIcon(dirs, filepath.Join(root, "pixmaps", "vieux.xpm")); filepath.Ext(got) != ".png" {
		t.Errorf("un .xpm doit être remplacé par le .png du même nom : %s", got)
	}
	if resolveIcon(dirs, "inconnu") != "" {
		t.Error("icône inconnue : chemin vide attendu")
	}
}

func TestExecProgram(t *testing.T) {
	cases := map[string]string{
		"/usr/bin/code --unity-launch %F": "/usr/bin/code",
		"env A=1 B=2 /opt/app/run %u":     "/opt/app/run",
		`"/usr/games/steam" %U`:           "/usr/games/steam",
	}
	for in, want := range cases {
		if got := execProgram(in); got != want {
			t.Errorf("execProgram(%q) = %q, attendu %q", in, got, want)
		}
	}
}
