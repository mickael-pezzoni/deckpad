// Package registry tient la liste des agents deckpad vus sur le réseau.
package registry

import (
	"cmp"
	"slices"
	"sync"
	"time"
)

// Agent est un PC qui fait tourner deckpad.
type Agent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IPs       []string  `json:"ips"`
	Port      int       `json:"port"`                // HTTP
	HTTPSPort int       `json:"httpsPort,omitempty"` // 0 si HTTPS désactivé
	Version   string    `json:"version"`
	LastSeen  time.Time `json:"lastSeen"`
}

type Registry struct {
	now func() time.Time

	mu     sync.Mutex
	agents map[string]Agent
}

func New() *Registry {
	return &Registry{now: time.Now, agents: map[string]Agent{}}
}

// Seen enregistre (ou met à jour) un agent qui vient de s'annoncer.
func (r *Registry) Seen(a Agent) {
	if a.ID == "" {
		return
	}
	a.LastSeen = r.now()
	r.mu.Lock()
	r.agents[a.ID] = a
	r.mu.Unlock()
}

// Prune retire les agents muets depuis plus de maxAge (PC éteint, agent fermé).
func (r *Registry) Prune(maxAge time.Duration) {
	limit := r.now().Add(-maxAge)
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, a := range r.agents {
		if a.LastSeen.Before(limit) {
			delete(r.agents, id)
		}
	}
}

// List renvoie les agents triés par nom.
func (r *Registry) List() []Agent {
	r.mu.Lock()
	list := make([]Agent, 0, len(r.agents))
	for _, a := range r.agents {
		list = append(list, a)
	}
	r.mu.Unlock()
	slices.SortFunc(list, func(a, b Agent) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	return list
}
