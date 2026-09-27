package network

import (
	"context"
	"testing"
)

func TestKindOf(t *testing.T) {
	cases := map[string]string{
		"Wi-Fi":      "wifi",
		"Wi-Fi 2":    "wifi",
		"wlp3s0":     "wifi",
		"Ethernet":   "ethernet",
		"Ethernet 2": "ethernet",
		"enp4s0":     "ethernet",
		"eth0":       "ethernet",
		"tun0":       "",
	}
	for in, want := range cases {
		if got := kindOf(in); got != want {
			t.Errorf("kindOf(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestCollectRates(t *testing.T) {
	m := NewMonitor()
	ctx := context.Background()
	s, err := m.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.RxRate != 0 || s.TxRate != 0 {
		t.Error("pas de débit sans mesure précédente")
	}
	s, err = m.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.RxRate < 0 || s.TxRate < 0 {
		t.Errorf("débit négatif : %+v", s)
	}
}
