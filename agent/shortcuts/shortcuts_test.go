package shortcuts

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestOpenWithoutFileGivesDefaults(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "shortcuts.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.List()) == 0 {
		t.Fatal("aucun raccourci par défaut")
	}
}

func TestDefaultsAreValid(t *testing.T) {
	for _, sc := range defaults() {
		if _, err := clean(sc); err != nil {
			t.Errorf("%s : %v", sc.ID, err)
		}
	}
}

func TestReplaceSavesAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deckpad", "shortcuts.json")
	s, _ := Open(path)
	list, err := s.Replace([]Shortcut{
		{Label: " OBS ", Kind: KindLaunch, Command: "obs", Keys: []string{"a"}},
		{Label: "Copier", Kind: KindKeys, Keys: []string{"C", "shift", "ctrl"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if list[0].ID == "" || list[0].ID == list[1].ID {
		t.Errorf("identifiants : %q %q", list[0].ID, list[1].ID)
	}
	if list[0].Label != "OBS" || list[0].Keys != nil {
		t.Errorf("non nettoyé : %+v", list[0])
	}
	if want := []string{"ctrl", "shift", "c"}; !slices.Equal(list[1].Keys, want) {
		t.Errorf("touches %v, attendu %v", list[1].Keys, want)
	}

	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.List(); len(got) != 2 || got[1].ID != list[1].ID {
		t.Errorf("rechargé : %+v", got)
	}
}

func TestReplaceRejectsInvalid(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "shortcuts.json"))
	for name, sc := range map[string]Shortcut{
		"sans nom":          {Kind: KindOpen, Target: "~"},
		"type inconnu":      {Label: "x", Kind: "format"},
		"touche inconnue":   {Label: "x", Kind: KindKeys, Keys: []string{"ctrl", "zz"}},
		"que modificateurs": {Label: "x", Kind: KindKeys, Keys: []string{"ctrl", "alt"}},
		"deux touches":      {Label: "x", Kind: KindKeys, Keys: []string{"a", "b"}},
		"commande vide":     {Label: "x", Kind: KindLaunch, Command: "  "},
	} {
		if _, err := s.Replace([]Shortcut{sc}); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s : attendu ErrInvalid, obtenu %v", name, err)
		}
	}
	if len(s.List()) != len(defaults()) {
		t.Error("une liste invalide ne doit rien changer")
	}
}

func TestRunUnknown(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "shortcuts.json"))
	if err := s.Run("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("attendu ErrNotFound, obtenu %v", err)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := expandHome("~"); got != home {
		t.Errorf("~ → %q", got)
	}
	if got := expandHome("~/Images"); got != filepath.Join(home, "Images") {
		t.Errorf("~/Images → %q", got)
	}
	if got := expandHome("https://x.fr/~a"); got != "https://x.fr/~a" {
		t.Errorf("adresse modifiée : %q", got)
	}
}
