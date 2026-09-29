//go:build !windows

package files

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseUserDirs(t *testing.T) {
	dirs := parseUserDirs(bufio.NewScanner(strings.NewReader(`# commentaire
XDG_DESKTOP_DIR="$HOME/Bureau"
XDG_DOWNLOAD_DIR="$HOME/Téléchargements"
`)), "/home/micka")
	if dirs["DESKTOP"] != "/home/micka/Bureau" || dirs["DOWNLOAD"] != "/home/micka/Téléchargements" {
		t.Errorf("dossiers = %v", dirs)
	}
}

func TestParseXBEL(t *testing.T) {
	items := parseXBEL([]byte(`<?xml version="1.0"?>
<xbel version="1.0">
  <bookmark href="file:///home/micka/Documents/Mes%20notes.txt" added="2026-09-01T10:00:00Z" modified="2026-09-02T10:00:00Z" visited="2026-09-03T10:00:00.123456Z"/>
  <bookmark href="https://example.com/" modified="2026-09-02T10:00:00Z"/>
</xbel>`))
	if len(items) != 1 || items[0].path != "/home/micka/Documents/Mes notes.txt" {
		t.Fatalf("items = %+v", items)
	}
	if items[0].used.Day() != 3 {
		t.Errorf("date = %v, attendu la plus récente", items[0].used)
	}
}
