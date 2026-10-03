package notify

import (
	"os"
	"testing"
	"time"
)

// Les tests n'affichent rien : show est remplacé une fois pour toutes (la file
// continue d'être vidée après la fin d'un test).
var (
	shown   = make(chan Message, 1)
	release = make(chan struct{})
)

func TestMain(m *testing.M) {
	show = func(m Message) error {
		shown <- m
		<-release
		return nil
	}
	os.Exit(m.Run())
}

func TestExcerpt(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"bonjour", "bonjour"},
		{" deux\nmots\t", "deux mots"},
		{"0123456789", "0123456789"},
		{"0123456789ab", "012345678…"},
		{"éééééééééééé", "ééééééééé…"},
	} {
		if got := Excerpt(c.in, 10); got != c.want {
			t.Errorf("Excerpt(%q) = %q, attendu %q", c.in, got, c.want)
		}
	}
}

func TestSendDoesNotBlock(t *testing.T) {
	Send("Fichier reçu", "a.txt")
	select {
	case m := <-shown:
		if m.Title != "Fichier reçu" || m.Body != "a.txt" {
			t.Fatalf("message %+v", m)
		}
	case <-time.After(time.Second):
		t.Fatal("notification non affichée")
	}
	// La notification affichée bloque : les suivantes s'accumulent puis sont
	// abandonnées, sans jamais bloquer l'appelant.
	done := make(chan struct{})
	go func() {
		for range 50 {
			Send("x", "")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Send bloque")
	}
	close(release)
}
