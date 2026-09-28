package shortcuts

import "testing"

func TestWebURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://exemple.fr/page?q=1": "https://exemple.fr/page?q=1",
		"  http://192.168.1.10:8080 ": "http://192.168.1.10:8080",
		"exemple.fr/page":             "https://exemple.fr/page",
		"youtube.com":                 "https://youtube.com",
	} {
		if got, err := webURL(in); err != nil || got != want {
			t.Errorf("webURL(%q) = %q, %v ; attendu %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "bonjour", "C:\\Windows\\notepad.exe", "file:///etc/passwd", "calc.exe arg", "javascript:alert(1)", "ftp://exemple.fr"} {
		if got, err := webURL(in); err == nil {
			t.Errorf("webURL(%q) accepté : %q", in, got)
		}
	}
}

func TestTypeTextRefuses(t *testing.T) {
	if err := TypeText(""); err == nil {
		t.Error("texte vide accepté")
	}
	long := make([]rune, maxTyped+1)
	for i := range long {
		long[i] = 'a'
	}
	if err := TypeText(string(long)); err != ErrLongText {
		t.Errorf("texte trop long : %v", err)
	}
}
