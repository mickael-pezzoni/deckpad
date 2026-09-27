// Package stats collecte les stats d'utilisation (page « Stats »).
package stats

import (
	"context"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type Snapshot struct {
	CPU      float64 `json:"cpu"` // % toutes cœurs confondus
	RAMUsed  uint64  `json:"ramUsed"`
	RAMTotal uint64  `json:"ramTotal"`
	GPU      *GPU    `json:"gpu"` // nil si aucun GPU lisible
	FPS      *int    `json:"fps"` // nil tant que la mesure n'est pas branchée
}

type GPU struct {
	Name     string  `json:"name"`
	Usage    float64 `json:"usage"` // %
	MemUsed  uint64  `json:"memUsed"`
	MemTotal uint64  `json:"memTotal"`
	Temp     float64 `json:"temp"` // °C
}

// Collect lit l'état courant. Le CPU est mesuré depuis l'appel précédent.
func Collect(ctx context.Context) (Snapshot, error) {
	var s Snapshot

	pct, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return s, err
	}
	if len(pct) > 0 {
		s.CPU = pct[0]
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return s, err
	}
	s.RAMUsed, s.RAMTotal = vm.Used, vm.Total

	s.GPU = readGPU(ctx)
	return s, nil
}
