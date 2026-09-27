package stats

import (
	"log"
	"sync"
)

var logged sync.Map

// logOnce écrit un message de diagnostic dans la console de l'agent, une seule fois.
func logOnce(format string, args ...any) {
	if _, dup := logged.LoadOrStore(format, true); !dup {
		log.Printf("GPU : "+format, args...)
	}
}
