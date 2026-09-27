package stats

import "testing"

func TestParseSMI(t *testing.T) {
	g := parseSMI("NVIDIA GeForce RTX 4070, 35, 2048, 12282, 54\n")
	if g == nil {
		t.Fatal("GPU attendu")
	}
	if g.Name != "NVIDIA GeForce RTX 4070" || g.Usage != 35 || g.Temp != 54 {
		t.Errorf("mauvaise lecture : %+v", g)
	}
	if g.MemUsed != 2048<<20 || g.MemTotal != 12282<<20 {
		t.Errorf("mémoire : %d / %d", g.MemUsed, g.MemTotal)
	}
	if parseSMI("") != nil || parseSMI("[N/A]") != nil {
		t.Error("une sortie invalide doit donner nil")
	}
}
