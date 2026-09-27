package stats

import (
	"context"
	"log"
	"sync"
	"time"
)

// Hub mesure une fois par intervalle et diffuse à toutes les tablettes connectées.
// Il ne tourne que tant qu'au moins un abonné est présent.
type Hub struct {
	interval time.Duration

	mu     sync.Mutex
	subs   map[chan Snapshot]struct{}
	cancel context.CancelFunc
}

func NewHub(interval time.Duration) *Hub {
	return &Hub{interval: interval, subs: map[chan Snapshot]struct{}{}}
}

// Subscribe renvoie un canal de mesures et la fonction pour se désabonner.
func (h *Hub) Subscribe() (<-chan Snapshot, func()) {
	ch := make(chan Snapshot, 1)

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

func (h *Hub) run(ctx context.Context) {
	Collect(ctx) // amorce la mesure CPU
	t := time.NewTicker(h.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		s, err := Collect(ctx)
		if err != nil {
			log.Printf("stats : %v", err)
			continue
		}
		h.mu.Lock()
		for ch := range h.subs {
			select {
			case ch <- s:
			default: // client lent : il aura la suivante
			}
		}
		h.mu.Unlock()
	}
}
