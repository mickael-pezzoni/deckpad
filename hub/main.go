// deckpad hub : serveur central, hébergé à la maison, qui trouve les agents
// deckpad du réseau local et les liste.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/mickael-pezzoni/deckpad/hub/discovery"
	"github.com/mickael-pezzoni/deckpad/hub/registry"
	"github.com/mickael-pezzoni/deckpad/hub/web"
)

func main() {
	addr := flag.String("addr", ":8430", "adresse d'écoute (ex: :8430)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	reg := registry.New()
	go discovery.Run(ctx, reg)

	srv := &http.Server{Addr: *addr, Handler: newHandler(reg)}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	log.Printf("deckpad hub sur %s", *addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func newHandler(reg *registry.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/agents", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reg.List())
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(web.Index)
	})
	return mux
}
