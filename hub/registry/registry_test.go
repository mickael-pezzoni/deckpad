package registry

import (
	"testing"
	"time"
)

func TestSeenListPrune(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	r := New()
	r.now = func() time.Time { return now }

	r.Seen(Agent{ID: "b", Name: "Salon"})
	r.Seen(Agent{ID: "a", Name: "Gaming"})
	r.Seen(Agent{Name: "sans id"}) // ignoré

	now = now.Add(30 * time.Second)
	r.Seen(Agent{ID: "a", Name: "Gaming", IPs: []string{"192.168.1.20"}}) // même PC, nouvelle IP

	list := r.List()
	if len(list) != 2 || list[0].ID != "a" || list[1].ID != "b" {
		t.Fatalf("List = %+v", list)
	}
	if list[0].IPs[0] != "192.168.1.20" {
		t.Errorf("IP non mise à jour : %+v", list[0])
	}

	now = now.Add(40 * time.Second) // b muet depuis 70 s, a depuis 40 s
	r.Prune(time.Minute)
	if list := r.List(); len(list) != 1 || list[0].ID != "a" {
		t.Fatalf("après Prune : %+v", list)
	}
}
