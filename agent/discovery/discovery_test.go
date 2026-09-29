package discovery

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadIDStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "agent-id")
	first, err := LoadID(path)
	if err != nil || len(first) != 16 {
		t.Fatalf("LoadID = %q, %v", first, err)
	}
	again, err := LoadID(path)
	if err != nil || again != first {
		t.Fatalf("second LoadID = %q, %v ; want %q", again, err, first)
	}
}

func TestLoadIDEmptyFileRegenerates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-id")
	if err := os.WriteFile(path, []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	id, err := LoadID(path)
	if err != nil || id == "" {
		t.Fatalf("LoadID = %q, %v", id, err)
	}
}

func TestText(t *testing.T) {
	if got := Text("abc"); !slices.Equal(got, []string{"id=abc", "v=2"}) {
		t.Errorf("Text = %v", got)
	}
}
