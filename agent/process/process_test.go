package process

import (
	"context"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestIsVisible(t *testing.T) {
	l := &Lister{me: `PC\micka`}
	cases := []struct {
		name, owner string
		want        bool
	}{
		{"game.exe", `PC\micka`, true},
		{"game.exe", `pc\MICKA`, true},
		{"svchost.exe", `PC\micka`, false},
		{"SvcHost.exe", `PC\micka`, false},
		{"lsass.exe", `NT AUTHORITY\SYSTEM`, false},
		{"lsass.exe", "", false},
	}
	for _, c := range cases {
		if got := l.isVisible(c.name, c.owner); got != c.want {
			t.Errorf("isVisible(%q, %q) = %v", c.name, c.owner, got)
		}
	}
}

func TestListAndKill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("utilise sleep")
	}
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	defer cmd.Process.Kill()

	ctx := context.Background()
	l := NewLister()
	apps, err := l.Apps(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range apps {
		if a.Name == "sleep" {
			found = true
		}
		if a.Name == "" || a.Count == 0 {
			t.Errorf("application invalide : %+v", a)
		}
	}
	if !found {
		t.Fatal("le processus sleep devrait être visible")
	}
	if _, err := l.Kill(ctx, "introuvable"); err != ErrNotFound {
		t.Errorf("attendu ErrNotFound, obtenu %v", err)
	}
	if n, err := l.Kill(ctx, "sleep"); err != nil || n < 1 {
		t.Fatalf("Kill : %d, %v", n, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("le processus n'a pas été fermé")
	}
}
