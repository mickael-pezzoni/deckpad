package sysinfo

import (
	"context"
	"os/user"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// uptime corrige la durée d'allumage de Windows. Avec le « démarrage rapide »
// (activé par défaut), « Arrêter » met le noyau en veille prolongée au lieu de
// l'éteindre : le compteur système continue sur plusieurs jours. La session de
// l'utilisateur, elle, est bien refermée à chaque arrêt : on prend donc le plus
// ancien de ses processus comme heure d'allumage.
func uptime(ctx context.Context, hostUptime uint64) uint64 {
	start, ok := sessionStart(ctx)
	if !ok {
		return hostUptime
	}
	since := uint64(time.Since(start).Seconds())
	return min(since, hostUptime)
}

func sessionStart(ctx context.Context) (time.Time, bool) {
	me, err := user.Current()
	if err != nil {
		return time.Time{}, false
	}
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return time.Time{}, false
	}
	var earliest int64
	for _, p := range procs {
		owner, err := p.UsernameWithContext(ctx)
		if err != nil || !strings.EqualFold(owner, me.Username) {
			continue
		}
		ms, err := p.CreateTimeWithContext(ctx)
		if err != nil || ms <= 0 {
			continue
		}
		if earliest == 0 || ms < earliest {
			earliest = ms
		}
	}
	if earliest == 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(earliest), true
}
