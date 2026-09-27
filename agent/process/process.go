// Package process liste les applications de l'utilisateur et permet de les fermer
// (page « Processus »). Les processus du système et des autres comptes sont masqués.
package process

import (
	"context"
	"errors"
	"os"
	"os/user"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/shirou/gopsutil/v4/process"
)

// App regroupe les processus d'un même programme, comme le Gestionnaire des tâches.
type App struct {
	Name  string  `json:"name"`
	CPU   float64 `json:"cpu"` // % de la machine entière
	RAM   uint64  `json:"ram"`
	Count int     `json:"count"`
}

type entry struct {
	proc    *process.Process
	name    string
	visible bool
}

// Lister garde les processus d'une mesure à l'autre : le CPU se calcule par différence.
type Lister struct {
	mu      sync.Mutex
	known   map[int32]*entry
	me      string
	ncpu    float64
	selfPID int32
}

func NewLister() *Lister {
	l := &Lister{known: map[int32]*entry{}, ncpu: float64(runtime.NumCPU()), selfPID: int32(os.Getpid())}
	if u, err := user.Current(); err == nil {
		l.me = u.Username
	}
	return l
}

// Apps renvoie les applications visibles, les plus gourmandes en CPU d'abord.
func (l *Lister) Apps(ctx context.Context) ([]App, error) {
	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	alive := make(map[int32]bool, len(pids))
	byName := map[string]*App{}
	for _, pid := range pids {
		alive[pid] = true
		e := l.entry(ctx, pid)
		if e == nil || !e.visible {
			continue
		}
		cpu, err := e.proc.PercentWithContext(ctx, 0)
		if err != nil {
			continue
		}
		var rss uint64
		if m, err := e.proc.MemoryInfoWithContext(ctx); err == nil {
			rss = m.RSS
		}
		a := byName[e.name]
		if a == nil {
			a = &App{Name: e.name}
			byName[e.name] = a
		}
		a.CPU += cpu / l.ncpu
		a.RAM += rss
		a.Count++
	}
	for pid := range l.known {
		if !alive[pid] {
			delete(l.known, pid)
		}
	}

	apps := make([]App, 0, len(byName))
	for _, a := range byName {
		apps = append(apps, *a)
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].CPU != apps[j].CPU {
			return apps[i].CPU > apps[j].CPU
		}
		return apps[i].RAM > apps[j].RAM
	})
	return apps, nil
}

// entry lit une seule fois le nom et le propriétaire d'un processus.
func (l *Lister) entry(ctx context.Context, pid int32) *entry {
	if e, ok := l.known[pid]; ok {
		return e
	}
	p, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		return nil
	}
	name, _ := p.NameWithContext(ctx)
	owner, _ := p.UsernameWithContext(ctx) // échoue pour les processus système : ils restent masqués
	e := &entry{proc: p, name: name, visible: pid != l.selfPID && l.isVisible(name, owner)}
	l.known[pid] = e
	return e
}

func (l *Lister) isVisible(name, owner string) bool {
	if name == "" || l.me == "" || !strings.EqualFold(owner, l.me) {
		return false
	}
	return !hidden[strings.ToLower(name)]
}

// ErrNotFound : aucune application visible ne porte ce nom.
var ErrNotFound = errors.New("application introuvable")

// Kill ferme tous les processus visibles portant ce nom et renvoie combien ont été fermés.
func (l *Lister) Kill(ctx context.Context, name string) (int, error) {
	l.mu.Lock()
	var targets []*process.Process
	for _, e := range l.known {
		if e.visible && e.name == name {
			targets = append(targets, e.proc)
		}
	}
	l.mu.Unlock()

	if len(targets) == 0 {
		return 0, ErrNotFound
	}
	killed := 0
	var lastErr error
	for _, p := range targets {
		if err := p.KillWithContext(ctx); err != nil {
			lastErr = err
			continue
		}
		killed++
	}
	if killed == 0 {
		return 0, lastErr
	}
	return killed, nil
}
