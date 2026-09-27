package process

import (
	"context"
	"errors"
	"sync"
)

// ErrNoIcon : pas d'icône disponible pour ce programme (ou pas sur ce système).
var ErrNoIcon = errors.New("icône indisponible")

const iconSize = 64

// Icon est une image prête à envoyer (PNG ou SVG).
type Icon struct {
	Data        []byte
	ContentType string
}

var (
	iconMu    sync.Mutex
	iconCache = map[string]*Icon{} // chemin de l'exe → icône (nil si échec)
)

// Icon renvoie l'icône du programme. Chaque système a sa source :
// l'exécutable lui-même sous Windows, les fichiers .desktop sous Linux.
func (l *Lister) Icon(ctx context.Context, name string) (*Icon, error) {
	path, err := l.exePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return IconOf(path, name)
}

// IconOf renvoie l'icône d'un exécutable déjà connu (ex. une appli qui joue du son).
func IconOf(exePath, name string) (*Icon, error) {
	iconMu.Lock()
	defer iconMu.Unlock()
	if ic, ok := iconCache[exePath]; ok {
		if ic == nil {
			return nil, ErrNoIcon
		}
		return ic, nil
	}
	ic, err := loadIcon(exePath, name)
	iconCache[exePath] = ic // on retient aussi les échecs pour ne pas réessayer en boucle
	if err != nil || ic == nil {
		iconCache[exePath] = nil
		return nil, ErrNoIcon
	}
	return ic, nil
}

func (l *Lister) exePath(ctx context.Context, name string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.known {
		if !e.visible || e.name != name {
			continue
		}
		if p, err := e.proc.ExeWithContext(ctx); err == nil && p != "" {
			return p, nil
		}
	}
	return "", ErrNotFound
}
