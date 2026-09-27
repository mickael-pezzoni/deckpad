// Package live diffuse des mesures périodiques à toutes les tablettes connectées.
package live

import (
	"context"
	"log"
	"sync"
	"time"
)

// Hub appelle collect à chaque intervalle et diffuse le résultat aux abonnés.
// Il ne tourne que tant qu'au moins un abonné est présent.
type Hub[T any] struct {
	interval time.Duration
	collect  func(context.Context) (T, error)

	mu     sync.Mutex
	subs   map[chan T]struct{}
	cancel context.CancelFunc
}

func NewHub[T any](interval time.Duration, collect func(context.Context) (T, error)) *Hub[T] {
	return &Hub[T]{interval: interval, collect: collect, subs: map[chan T]struct{}{}}
}

// Subscribe renvoie un canal de mesures et la fonction pour se désabonner.
func (h *Hub[T]) Subscribe() (<-chan T, func()) {
	ch := make(chan T, 1)

	h.mu.Lock()
	h.subs[ch] = struct{}{}
	if h.cancel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		h.cancel = cancel
		go h.run(ctx)
	}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.subs, ch)
		if len(h.subs) == 0 && h.cancel != nil {
			h.cancel()
			h.cancel = nil
		}
	}
}

func (h *Hub[T]) run(ctx context.Context) {
	h.collect(ctx) // amorce les mesures calculées par différence (CPU)
	t := time.NewTicker(h.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		v, err := h.collect(ctx)
		if err != nil {
			log.Printf("mesure : %v", err)
			continue
		}
		h.mu.Lock()
		for ch := range h.subs {
			select {
			case ch <- v:
			default: // client lent : il aura la suivante
			}
		}
		h.mu.Unlock()
	}
}
