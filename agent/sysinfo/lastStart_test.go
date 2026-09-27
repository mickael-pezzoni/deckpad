package sysinfo

import (
	"testing"
	"time"
)

func TestParseEventTime(t *testing.T) {
	xml := `<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System>` +
		`<Provider Name='Microsoft-Windows-Kernel-Boot'/><EventID>27</EventID>` +
		`<TimeCreated SystemTime='2026-09-26T21:04:12.5271234Z'/></System></Event>`
	got, err := parseEventTime(xml)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 26, 21, 4, 12, 527123400, time.UTC)
	if !got.Equal(want) {
		t.Errorf("obtenu %v, attendu %v", got, want)
	}
	if _, err := parseEventTime(""); err == nil {
		t.Error("une sortie vide doit renvoyer une erreur")
	}
}
