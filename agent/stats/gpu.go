package stats

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// nvidia-smi est livré avec le pilote NVIDIA, sous Windows comme sous Linux.
// Les autres GPU ne sont pas encore pris en charge.
var (
	smiOnce sync.Once
	smiPath string
)

func readGPU(ctx context.Context) *GPU {
	smiOnce.Do(func() { smiPath, _ = exec.LookPath("nvidia-smi") })
	if smiPath == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, smiPath,
		"--query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu",
		"--format=csv,noheader,nounits")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return parseSMI(string(out))
}

// parseSMI lit la première ligne, ex : "NVIDIA GeForce RTX 4070, 35, 2048, 12282, 54".
func parseSMI(out string) *GPU {
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	f := strings.Split(line, ",")
	if len(f) != 5 {
		return nil
	}
	num := func(i int) float64 {
		v, _ := strconv.ParseFloat(strings.TrimSpace(f[i]), 64)
		return v
	}
	const mib = 1 << 20
	return &GPU{
		Name:     strings.TrimSpace(f[0]),
		Usage:    num(1),
		MemUsed:  uint64(num(2)) * mib,
		MemTotal: uint64(num(3)) * mib,
		Temp:     num(4),
	}
}
