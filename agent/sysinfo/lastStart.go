package sysinfo

import (
	"errors"
	"regexp"
	"time"
)

// lastStartQuery sélectionne dans le journal Système les événements qui marquent
// une mise en route du PC :
//   - Kernel-Boot 27          : chaque démarrage, y compris le « démarrage rapide »
//   - Kernel-General 12       : démarrage complet du système
//   - Power-Troubleshooter 1  : réveil après une veille ou une veille prolongée
const lastStartQuery = `*[System[` +
	`(Provider[@Name='Microsoft-Windows-Kernel-Boot'] and EventID=27) or ` +
	`(Provider[@Name='Microsoft-Windows-Kernel-General'] and EventID=12) or ` +
	`(Provider[@Name='Microsoft-Windows-Power-Troubleshooter'] and EventID=1)]]`

var systemTimeRe = regexp.MustCompile(`SystemTime=['"]([^'"]+)['"]`)

// parseEventTime lit la date d'un événement exporté en XML par wevtutil.
func parseEventTime(xml string) (time.Time, error) {
	m := systemTimeRe.FindStringSubmatch(xml)
	if m == nil {
		return time.Time{}, errors.New("aucun événement de démarrage")
	}
	return time.Parse(time.RFC3339Nano, m[1])
}
