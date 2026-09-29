package files

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unicode/utf16"
)

// makeLnk fabrique un raccourci minimal : en-tête, liste d'ID à sauter, LinkInfo.
func makeLnk(ansi string, unicode string) []byte {
	b := make([]byte, 0x4C)
	binary.LittleEndian.PutUint32(b, 0x4C)
	binary.LittleEndian.PutUint32(b[0x14:], 0x1|0x2)
	b = append(b, 4, 0, 0xAA, 0xBB, 0xCC, 0xDD) // liste d'ID de 4 octets

	header := uint32(0x1C)
	if unicode != "" {
		header = 0x24
	}
	info := make([]byte, header)
	binary.LittleEndian.PutUint32(info[4:], header)
	binary.LittleEndian.PutUint32(info[8:], 0x1)
	binary.LittleEndian.PutUint32(info[16:], uint32(len(info)))
	info = append(append(info, ansi...), 0)
	binary.LittleEndian.PutUint32(info[24:], uint32(len(info)))
	info = append(info, 0) // suffixe vide
	if unicode != "" {
		binary.LittleEndian.PutUint32(info[28:], uint32(len(info)))
		for _, c := range utf16.Encode([]rune(unicode)) {
			info = binary.LittleEndian.AppendUint16(info, c)
		}
		info = append(info, 0, 0)
		binary.LittleEndian.PutUint32(info[32:], uint32(len(info)))
		info = append(info, 0, 0)
	}
	binary.LittleEndian.PutUint32(info, uint32(len(info)))
	return append(b, info...)
}

func TestLnkTarget(t *testing.T) {
	if got := lnkTarget(makeLnk(`C:\Users\micka\CV.pdf`, "")); got != `C:\Users\micka\CV.pdf` {
		t.Errorf("ANSI : %q", got)
	}
	if got := lnkTarget(makeLnk(`C:\?`, `C:\Users\micka\Été.jpg`)); got != `C:\Users\micka\Été.jpg` {
		t.Errorf("Unicode : %q", got)
	}
	for _, bad := range [][]byte{nil, make([]byte, 10), makeLnk("x", "")[:0x50]} {
		if got := lnkTarget(bad); got != "" {
			t.Errorf("raccourci invalide : %q", got)
		}
	}
}

func TestPickRecents(t *testing.T) {
	dir := t.TempDir()
	write := func(name string) string {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x"), 0o644)
		return p
	}
	old, recent := write("old.txt"), write("recent.txt")
	hidden := write(filepath.Join(".cache", "a.txt"))
	now := time.Now()
	got := pickRecents([]Drive{{Path: dir}}, []recentItem{
		{old, now.Add(-time.Hour)},
		{recent, now},
		{recent, now.Add(-2 * time.Hour)}, // doublon
		{filepath.Join(dir, "supprime.txt"), now},
		{dir, now},    // dossier
		{hidden, now}, // dans un dossier caché (Linux)
		{"/ailleurs/x.txt", now},
	})
	want := []string{"recent.txt", "old.txt"}
	if !hiddenPath(hidden) {
		want = []string{"recent.txt", "a.txt", "old.txt"}
	}
	if len(got) != len(want) {
		t.Fatalf("récents = %+v", got)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("récents[%d] = %s, attendu %s", i, got[i].Name, name)
		}
	}
	if got[0].Folder != dir {
		t.Errorf("dossier = %q", got[0].Folder)
	}
}
