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
	addr := flag.String("addr", ":8421", "adresse d'écoute de l'API HTTPS, appelée par le hub")
	windowAddr := flag.String("window-addr", "127.0.0.1:8420", "adresse locale de la fenêtre du code d'appairage")
	announce := flag.Bool("announce", true, "s'annoncer sur le réseau local (mDNS) pour le hub")
	flag.Parse()

	path, err := auth.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	store, err := auth.Open(path, func(code string) {
		log.Printf("code d'appairage : %s", code)
		if server.PairWindowOpen() {
			return
		}
		if err := window.Open("http://" + *windowAddr + "/pair-code"); err != nil {
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

	dir, err := tlscert.DefaultDir()
	if err != nil {
		log.Fatal(err)
	}
	cert, err := tlscert.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	handler := server.New(store, keys)

	// Fenêtre du code : servie au PC seul, en HTTP (le navigateur ne connaît pas le certificat).
	winLn, err := net.Listen("tcp", *windowAddr)
	if err != nil {
		log.Fatal(err)
	}
	go func() { log.Printf("fenêtre d'appairage arrêtée : %v", http.Serve(winLn, handler)) }()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	log.Printf("deckpad en écoute sur le port %s, en attente du hub", port)
	// L'annonce dure autant que l'agent.
	if *announce {
		startAnnounce(port)
	}
	log.Fatal(http.Serve(tls.NewListener(ln, tlscert.TLSConfig(cert)), handler))
}

// startAnnounce publie l'agent en mDNS. En cas d'échec (pare-feu, pas de
// multicast), le hub ne le trouve pas tout seul, mais l'agent tourne quand même.
func startAnnounce(port string) *discovery.Announcer {
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
	a, err := discovery.Announce(name, id, p)
	if err != nil {
		log.Printf("annonce mDNS indisponible : %v", err)
		return nil
	}
	log.Printf("annoncé sur le réseau local (mDNS) comme %q", name)
	return a
}
