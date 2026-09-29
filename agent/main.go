package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/mickael-pezzoni/deckpad/agent/auth"
	"github.com/mickael-pezzoni/deckpad/agent/discovery"
	"github.com/mickael-pezzoni/deckpad/agent/server"
	"github.com/mickael-pezzoni/deckpad/agent/shortcuts"
	"github.com/mickael-pezzoni/deckpad/agent/tlscert"
	"github.com/mickael-pezzoni/deckpad/agent/window"
)

func main() {
	addr := flag.String("addr", ":8420", "adresse d'écoute (ex: :8420)")
	tlsAddr := flag.String("https-addr", ":8421", "adresse d'écoute HTTPS (vide pour désactiver)")
	announce := flag.Bool("announce", true, "s'annoncer sur le réseau local (mDNS) pour le serveur central")
	flag.Parse()

	_, port, err := net.SplitHostPort(*addr)
	if err != nil {
		port = "8420"
	}

	path, err := auth.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	store, err := auth.Open(path, func(code string) {
		log.Printf("code d'appairage : %s", code)
		if server.PairWindowOpen() {
			return
		}
		if err := window.Open("http://127.0.0.1:" + port + "/pair-code"); err != nil {
			log.Printf("fenêtre du code : %v", err)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	keysPath, err := shortcuts.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	keys, err := shortcuts.Open(keysPath)
	if err != nil {
		log.Fatal(err)
	}

	sec := &server.Secure{HTTPPort: port}
	var tlsLn net.Listener
	if *tlsAddr != "" {
		tlsLn, sec.TLSPort, sec.CA = listenTLS(*tlsAddr)
	}
	handler := server.New(store, keys, sec)
	if tlsLn != nil {
		go func() { log.Printf("HTTPS arrêté : %v", http.Serve(tlsLn, handler)) }()
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	logLocalURLs(port, sec.TLSPort)
	// L'annonce dure autant que l'agent.
	if *announce {
		startAnnounce(port, sec.TLSPort)
	}
	// Aucune tablette encore : on affiche tout de suite le QR code à scanner.
	if store.Empty() {
		if err := window.Open("http://127.0.0.1:" + port + "/pair-code"); err != nil {
			log.Printf("fenêtre d'appairage : %v", err)
		}
	}
	log.Fatal(http.Serve(ln, handler))
}

// listenTLS ouvre le port HTTPS. En cas d'échec, deckpad continue en HTTP seul.
func listenTLS(addr string) (net.Listener, string, *tlscert.Authority) {
	dir, err := tlscert.DefaultDir()
	if err == nil {
		var ca *tlscert.Authority
		if ca, err = tlscert.Open(dir); err == nil {
			var ln net.Listener
			if ln, err = net.Listen("tcp", addr); err == nil {
				_, port, _ := net.SplitHostPort(ln.Addr().String())
				return tls.NewListener(ln, ca.TLSConfig()), port, ca
			}
		}
	}
	log.Printf("HTTPS indisponible, HTTP seul : %v", err)
	return nil, "", nil
}

// logLocalURLs affiche les adresses à ouvrir depuis la tablette.
func logLocalURLs(port, tlsPort string) {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		log.Printf("deckpad disponible sur http://%s:%s", ipNet.IP, port)
		if tlsPort != "" {
			log.Printf("  et en sécurisé sur https://%s:%s", ipNet.IP, tlsPort)
		}
	}
}

// startAnnounce publie l'agent en mDNS. En cas d'échec (pare-feu, pas de
// multicast), deckpad marche comme avant : seul le serveur central ne le voit pas.
func startAnnounce(port, tlsPort string) *discovery.Announcer {
	p, err := strconv.Atoi(port)
	if err != nil {
		log.Printf("annonce mDNS : port %q invalide", port)
		return nil
	}
	idPath, err := discovery.DefaultIDPath()
	if err != nil {
		log.Printf("annonce mDNS : %v", err)
		return nil
	}
	id, err := discovery.LoadID(idPath)
	if err != nil {
		log.Printf("annonce mDNS : %v", err)
		return nil
	}
	name, err := os.Hostname()
	if err != nil || name == "" {
		name = "deckpad"
	}
	a, err := discovery.Announce(name, id, p, tlsPort)
	if err != nil {
		log.Printf("annonce mDNS indisponible : %v", err)
		return nil
	}
	log.Printf("annoncé sur le réseau local (mDNS) comme %q", name)
	return a
}
