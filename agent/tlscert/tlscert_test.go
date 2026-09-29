package tlscert

import (
	"bytes"
	"testing"
)

func TestOpenKeepsCertificate(t *testing.T) {
	dir := t.TempDir()
	a, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Le hub reconnaît le PC à l'empreinte de ce certificat : elle ne doit pas changer.
	if !bytes.Equal(a.Certificate[0], b.Certificate[0]) {
		t.Fatal("le certificat doit être gardé d'un lancement à l'autre")
	}
}
