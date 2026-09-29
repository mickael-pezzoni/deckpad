// deckpad hub : serveur central, hébergé à la maison. Il trouve les agents
// deckpad du réseau local, sert l'appli tablette et relaie ses requêtes au PC choisi.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mickael-pezzoni/deckpad/hub/agents"
	"github.com/mickael-pezzoni/deckpad/hub/auth"
	"github.com/mickael-pezzoni/deckpad/hub/discovery"
	"github.com/mickael-pezzoni/deckpad/hub/registry"
	"github.com/mickael-pezzoni/deckpad/hub/server"
	"github.com/mickael-pezzoni/deckpad/hub/tlscert"
)

func main() {
	addr := flag.String("addr", ":8430", "adresse d'écoute HTTP")
	tlsAddr := flag.String("https-addr", ":8431", "adresse d'écoute HTTPS (vide pour désactiver)")
	data := flag.String("data", defaultDataDir(), "dossier des données (appairages, certificat)")
	flag.Parse()

	tablets, err := auth.Open(filepath.Join(*data, "tablets.json"))
	if err != nil {
		log.Fatal(err)
	}
	pcs, err := agents.Open(filepath.Join(*data, "agents.json"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	reg := registry.New()
	go discovery.Run(ctx, reg)

	sec := &server.Secure{}
	var tlsLn net.Listener
	if *tlsAddr != "" {
		tlsLn, sec.TLSPort, sec.CA = listenTLS(*tlsAddr, *data)
	}
	handler := server.New(tablets, pcs, reg, sec)

	srv := &http.Server{Addr: *addr, Handler: handler}
	tlsSrv := &http.Server{Handler: handler}
	if tlsLn != nil {
		go func() {
			if err := tlsSrv.Serve(tlsLn); err != http.ErrServerClosed {
				log.Printf("HTTPS arrêté : %v", err)
			}
		}()
	}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
		_ = tlsSrv.Shutdown(context.Background())
	}()
	log.Printf("deckpad hub sur %s (données : %s)", *addr, *data)
	if sec.TLSPort != "" {
		log.Printf("  et en HTTPS sur le port %s", sec.TLSPort)
	}
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// defaultDataDir : %APPDATA%\deckpad-hub sous Windows, ~/.config/deckpad-hub sous Linux.
func defaultDataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "deckpad-hub"
	}
	return filepath.Join(dir, "deckpad-hub")
}

// listenTLS ouvre le port HTTPS. En cas d'échec, le hub continue en HTTP seul.
func listenTLS(addr, dir string) (net.Listener, string, *tlscert.Authority) {
	ca, err := tlscert.Open(dir)
	if err == nil {
		var ln net.Listener
		if ln, err = net.Listen("tcp", addr); err == nil {
			_, port, _ := net.SplitHostPort(ln.Addr().String())
			return tls.NewListener(ln, ca.TLSConfig()), port, ca
		}
	}
	log.Printf("HTTPS indisponible, HTTP seul : %v", err)
	return nil, "", nil
}
