package discovery

import (
	"net"
	"testing"

	"github.com/libp2p/zeroconf/v2"
)

func TestFromEntry(t *testing.T) {
	e := &zeroconf.ServiceEntry{
		ServiceRecord: zeroconf.ServiceRecord{Instance: `Mon\ PC`},
		Port:          8421,
		Text:          []string{"id=abc", "v=2"},
		AddrIPv4:      []net.IP{net.IPv4(192, 168, 1, 20)},
	}
	a, ok := FromEntry(e)
	if !ok || a.ID != "abc" || a.Name != "Mon PC" || a.Port != 8421 ||
		a.Version != "2" || len(a.IPs) != 1 || a.IPs[0] != "192.168.1.20" {
		t.Fatalf("FromEntry = %+v, %v", a, ok)
	}
}

func TestFromEntryRejects(t *testing.T) {
	noID := &zeroconf.ServiceEntry{Text: []string{"v=2"}, AddrIPv4: []net.IP{net.IPv4(192, 168, 1, 20)}}
	if _, ok := FromEntry(noID); ok {
		t.Error("agent sans id accepté")
	}
	noIP := &zeroconf.ServiceEntry{Text: []string{"id=abc", "v=2"}}
	if _, ok := FromEntry(noIP); ok {
		t.Error("agent sans IPv4 accepté")
	}
	old := &zeroconf.ServiceEntry{Text: []string{"id=abc", "v=1"}, AddrIPv4: []net.IP{net.IPv4(192, 168, 1, 20)}}
	if _, ok := FromEntry(old); ok {
		t.Error("ancien agent (v1) accepté")
	}
}
