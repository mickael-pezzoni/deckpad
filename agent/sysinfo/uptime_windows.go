package sysinfo

import (
	"context"
	"log"
	"os/exec"
	"syscall"
	"time"
)

// uptime corrige la durée d'allumage de Windows. Le compteur système ne repart
// pas à zéro avec le « démarrage rapide » (activé par défaut : « Arrêter » met le
// noyau en veille prolongée) ni après une mise en veille. On lit donc dans le
// journal Système le dernier démarrage ou réveil (voir lastStartQuery).
func uptime(ctx context.Context, hostUptime uint64) uint64 {
	start, err := lastStart(ctx)
	if err != nil {
		log.Printf("durée d'allumage : journal d'événements illisible (%v), compteur système utilisé", err)
		return hostUptime
	}
	since := uint64(max(0, time.Since(start).Seconds()))
	return min(since, hostUptime)
}

func lastStart(ctx context.Context) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "wevtutil", "qe", "System",
		"/q:"+lastStartQuery, "/c:1", "/rd:true", "/f:xml")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	return parseEventTime(string(out))
}
