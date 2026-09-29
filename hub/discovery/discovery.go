// Package discovery trouve les agents deckpad du réseau local en mDNS.
//
// Le type de service « _deckpad._tcp » et les clés TXT (id, v) sont
// partagés avec agent/discovery : les changer des deux côtés à la fois.
package discovery

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/zeroconf/v2"

	"github.com/mickael-pezzoni/deckpad/hub/registry"
)

const (
	Service = "_deckpad._tcp"
	Domain  = "local."

	// Une recherche dure scanFor, puis on attend jusqu'à every. Un agent absent
	// de plusieurs recherches de suite (maxAge) est retiré de la liste : un PC
	// coupé brutalement n'envoie pas d'au revoir.
	scanFor = 3 * time.Second
	every   = 8 * time.Second
	maxAge  = 30 * time.Second

	minVersion = 2
)

// Run cherche les agents jusqu'à l'arrêt de ctx et tient reg à jour.
func Run(ctx context.Context, reg *registry.Registry) {
	for {
		scan(ctx, reg)
		reg.Prune(maxAge)
		select {
		case <-ctx.Done():
			return
		case <-time.After(every - scanFor):
		}
	}
}

// scan fait une recherche. Chaque recherche repart de zéro, donc chaque agent
// présent y répond de nouveau et son heure de passage est rafraîchie.
func scan(ctx context.Context, reg *registry.Registry) {
	ctx, cancel := context.WithTimeout(ctx, scanFor)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for e := range entries {
			if a, ok := FromEntry(e); ok {
				reg.Seen(a)
			}
		}
	}()
	// Browse bloque jusqu'à la fin de ctx et ferme alors entries, sauf s'il
	// échoue avant d'avoir démarré : on le ferme nous-mêmes dans ce cas.
	// IPv4 seul : on n'utilise que ces adresses, et une machine (ou un conteneur)
	// sans IPv6 ferait sinon échouer toute la recherche.
	if err := zeroconf.Browse(ctx, Service, Domain, entries, zeroconf.SelectIPTraffic(zeroconf.IPv4)); err != nil {
		log.Printf("recherche mDNS : %v", err)
		closeIfOpen(entries)
	}
	<-done
}

func closeIfOpen(ch chan *zeroconf.ServiceEntry) {
	defer func() { _ = recover() }() // déjà fermé par Browse
	close(ch)
}

// FromEntry convertit une réponse mDNS en agent. ok est faux si ce n'est pas
// un agent deckpad utilisable : sans identifiant, sans adresse IPv4, ou trop
// ancien (avant la version 2, l'agent servait l'appli et non une API HTTPS).
func FromEntry(e *zeroconf.ServiceEntry) (a registry.Agent, ok bool) {
	txt := map[string]string{}
	for _, kv := range e.Text {
		k, v, _ := strings.Cut(kv, "=")
		txt[k] = v
	}
	a = registry.Agent{
		ID:      txt["id"],
		Name:    unescape(e.Instance),
		Port:    e.Port,
		Version: txt["v"],
	}
	for _, ip := range e.AddrIPv4 {
		a.IPs = append(a.IPs, ip.String())
	}
	v, _ := strconv.Atoi(a.Version)
	return a, a.ID != "" && len(a.IPs) > 0 && v >= minVersion
}

// unescape retire les « \ » que DNS ajoute devant les espaces et les points
// des noms d'instance (« Mon\ PC » -> « Mon PC »).
func unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
