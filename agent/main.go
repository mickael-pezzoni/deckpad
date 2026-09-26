package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/mickael-pezzoni/deckpad/agent/server"
)

func main() {
	addr := flag.String("addr", ":8420", "adresse d'écoute (ex: :8420)")
	flag.Parse()

	logLocalURLs(*addr)
	log.Fatal(http.ListenAndServe(*addr, server.New()))
}

// logLocalURLs affiche les adresses à ouvrir depuis la tablette.
func logLocalURLs(addr string) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		port = "8420"
	}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		log.Printf("deckpad disponible sur http://%s:%s", ipNet.IP, port)
	}
}
