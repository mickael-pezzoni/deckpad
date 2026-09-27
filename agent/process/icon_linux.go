package process

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Sous Linux, les icônes des applications sont déclarées dans les fichiers
// .desktop (clé Icon=) et installées dans le thème « hicolor » ou /usr/share/pixmaps.
// On relie un processus à son .desktop par le nom de l'exécutable (Exec=, TryExec=),
// la classe de fenêtre (StartupWMClass=) ou le nom du fichier.

var (
	desktopOnce  sync.Once
	desktopIcons map[string]string // nom de programme (minuscules) → nom ou chemin d'icône
)

func loadIcon(exePath, name string) (*Icon, error) {
	dirs := dataDirs()
	desktopOnce.Do(func() { desktopIcons = indexDesktopFiles(dirs) })

	for _, key := range []string{strings.ToLower(filepath.Base(exePath)), strings.ToLower(name)} {
		if icon, ok := desktopIcons[key]; ok {
			if path := resolveIcon(dirs, icon); path != "" {
				return readIcon(path)
			}
		}
	}
	return nil, ErrNoIcon
}

// dataDirs suit la spécification XDG, plus les emplacements Flatpak et Snap.
func dataDirs() []string {
	var dirs []string
	home, _ := os.UserHomeDir()
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		dirs = append(dirs, d)
	} else if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "share"))
	}
	sys := os.Getenv("XDG_DATA_DIRS")
	if sys == "" {
		sys = "/usr/local/share:/usr/share"
	}
	dirs = append(dirs, filepath.SplitList(sys)...)
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "flatpak", "exports", "share"))
	}
	return append(dirs, "/var/lib/flatpak/exports/share", "/var/lib/snapd/desktop")
}

func indexDesktopFiles(dirs []string) map[string]string {
	index := map[string]string{}
	for _, d := range dirs {
		files, _ := filepath.Glob(filepath.Join(d, "applications", "*.desktop"))
		sub, _ := filepath.Glob(filepath.Join(d, "applications", "*", "*.desktop"))
		for _, f := range append(files, sub...) {
			icon, keys := parseDesktopFile(f)
			if icon == "" {
				continue
			}
			for _, k := range keys {
				if _, seen := index[k]; !seen { // le premier dossier (utilisateur) a priorité
					index[k] = icon
				}
			}
		}
	}
	return index
}

// parseDesktopFile renvoie l'icône et les noms de programme qu'elle représente.
func parseDesktopFile(path string) (icon string, keys []string) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil
	}
	defer f.Close()

	add := func(k string) {
		if k = strings.ToLower(strings.TrimSpace(k)); k != "" {
			keys = append(keys, k)
		}
	}
	add(strings.TrimSuffix(filepath.Base(path), ".desktop"))

	inEntry := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !inEntry || !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "Icon":
			icon = strings.TrimSpace(v)
		case "Exec", "TryExec":
			add(filepath.Base(execProgram(v)))
		case "StartupWMClass":
			add(v)
		}
	}
	return icon, keys
}

// execProgram extrait le programme d'une ligne Exec=, en sautant « env VAR=… ».
func execProgram(exec string) string {
	fields := strings.Fields(strings.ReplaceAll(exec, `"`, ""))
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if filepath.Base(f) == "env" || strings.Contains(f, "=") {
			continue
		}
		return f
	}
	return ""
}

// resolveIcon trouve le fichier d'une icône, en préférant une taille proche de 64 px.
func resolveIcon(dirs []string, icon string) string {
	if filepath.IsAbs(icon) {
		if _, err := os.Stat(icon); err == nil && browserImage(icon) {
			return icon
		}
		// Ex. /usr/share/pixmaps/app.xpm : format illisible par le navigateur,
		// on cherche une version PNG ou SVG du même nom.
		icon = strings.TrimSuffix(filepath.Base(icon), filepath.Ext(icon))
	}
	sizes := []string{"64x64", "48x48", "128x128", "96x96", "256x256", "32x32"}
	var candidates []string
	for _, d := range dirs {
		for _, s := range sizes {
			candidates = append(candidates, filepath.Join(d, "icons", "hicolor", s, "apps", icon+".png"))
		}
		candidates = append(candidates, filepath.Join(d, "icons", "hicolor", "scalable", "apps", icon+".svg"))
		candidates = append(candidates, filepath.Join(d, "pixmaps", icon+".png"), filepath.Join(d, "pixmaps", icon+".svg"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func browserImage(path string) bool {
	return strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".svg")
}

func readIcon(path string) (*Icon, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ct := "image/png"
	if strings.HasSuffix(path, ".svg") {
		ct = "image/svg+xml"
	}
	return &Icon{Data: data, ContentType: ct}, nil
}
