package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/mickael-pezzoni/deckpad/agent/auth"
	"github.com/mickael-pezzoni/deckpad/agent/server"
	"github.com/mickael-pezzoni/deckpad/agent/shortcuts"
	"github.com/mickael-pezzoni/deckpad/agent/window"
)

func main() {
	addr := flag.String("addr", ":8420", "adresse d'écoute (ex: :8420)")
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

	logLocalURLs(port)
	log.Fatal(http.ListenAndServe(*addr, server.New(store, keys)))
}

// logLocalURLs affiche les adresses à ouvrir depuis la tablette.
func logLocalURLs(port string) {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		log.Printf("deckpad disponible sur http://%s:%s", ipNet.IP, port)
	}
}
