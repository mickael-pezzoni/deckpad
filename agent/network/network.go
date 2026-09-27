// Package network mesure la connexion du PC (page « Réseau ») : interface utilisée,
// IP locale, débit montant/descendant et latence vers Internet.
package network

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
)

type Snapshot struct {
	Interface string  `json:"interface"`
	Kind      string  `json:"kind"` // "wifi", "ethernet" ou "" si inconnu
	LocalIP   string  `json:"localIP"`
	RxRate    float64 `json:"rxRate"` // octets/s reçus
	TxRate    float64 `json:"txRate"` // octets/s envoyés
	LatencyMs *int    `json:"latencyMs"`
}

// Monitor garde la mesure précédente : le débit se calcule par différence.
type Monitor struct {
	mu        sync.Mutex
	lastAt    time.Time
	lastRx    uint64
	lastTx    uint64
	lastIface string

	latencyAt time.Time
	latency   *int
}

func NewMonitor() *Monitor { return &Monitor{} }

const latencyEvery = 5 * time.Second

func (m *Monitor) Collect(ctx context.Context) (Snapshot, error) {
	iface, ip := primaryInterface()
	s := Snapshot{Interface: iface, Kind: kindOf(iface), LocalIP: ip}

	counters, err := gnet.IOCountersWithContext(ctx, true)
	if err != nil {
		return s, err
	}
	var rx, tx uint64
	for _, c := range counters {
		if c.Name == iface {
			rx, tx = c.BytesRecv, c.BytesSent
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if iface == m.lastIface && !m.lastAt.IsZero() && rx >= m.lastRx && tx >= m.lastTx {
		dt := now.Sub(m.lastAt).Seconds()
		s.RxRate = float64(rx-m.lastRx) / dt
		s.TxRate = float64(tx-m.lastTx) / dt
	}
	m.lastAt, m.lastRx, m.lastTx, m.lastIface = now, rx, tx, iface

	if now.Sub(m.latencyAt) >= latencyEvery {
		m.latencyAt = now
		m.latency = measureLatency(ctx)
	}
	s.LatencyMs = m.latency
	return s, nil
}

// primaryInterface trouve l'interface qui sort vers Internet : on « connecte » une
// socket UDP (aucun paquet n'est envoyé) et on regarde l'adresse locale choisie.
func primaryInterface() (name, ip string) {
	conn, err := net.Dial("udp4", "1.1.1.1:80")
	if err != nil {
		return "", ""
	}
	local := conn.LocalAddr().(*net.UDPAddr).IP
	conn.Close()

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.Equal(local) {
				return i.Name, local.String()
			}
		}
	}
	return "", local.String()
}

// kindOf devine le type de connexion d'après le nom de l'interface
// (Windows : « Wi-Fi », « Ethernet » ; Linux : wlan0, wlp3s0, eth0, enp4s0…).
func kindOf(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "wi-fi"), strings.Contains(n, "wifi"), strings.Contains(n, "wlan"),
		strings.HasPrefix(n, "wl"), strings.Contains(n, "wireless"):
		return "wifi"
	case strings.Contains(n, "ethernet"), strings.HasPrefix(n, "eth"), strings.HasPrefix(n, "en"):
		return "ethernet"
	}
	return ""
}

// measureLatency chronomètre l'ouverture d'une connexion TCP vers un serveur public
// (le ping ICMP demanderait les droits administrateur).
func measureLatency(ctx context.Context) *int {
	ctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	start := time.Now()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", "1.1.1.1:443")
	if err != nil {
		return nil
	}
	conn.Close()
	ms := int(time.Since(start).Milliseconds())
	return &ms
}
