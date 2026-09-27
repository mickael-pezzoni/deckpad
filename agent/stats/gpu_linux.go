package stats

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// readPlatformGPU lit le pilote amdgpu via sysfs (charge, VRAM, température).
func readPlatformGPU() *GPU {
	return readAMDSysfs("/sys/class/drm")
}

func readAMDSysfs(drm string) *GPU {
	cards, _ := filepath.Glob(filepath.Join(drm, "card[0-9]*", "device"))
	var best *GPU
	for _, dev := range cards {
		busy, ok := readUint(filepath.Join(dev, "gpu_busy_percent"))
		if !ok {
			continue // pas un GPU AMD
		}
		g := &GPU{Name: amdName(dev), Usage: float64(busy)}
		g.MemUsed, _ = readUint(filepath.Join(dev, "mem_info_vram_used"))
		g.MemTotal, _ = readUint(filepath.Join(dev, "mem_info_vram_total"))
		if hw, _ := filepath.Glob(filepath.Join(dev, "hwmon", "hwmon*", "temp1_input")); len(hw) > 0 {
			if milli, ok := readUint(hw[0]); ok {
				t := float64(milli) / 1000
				g.Temp = &t
			}
		}
		// Avec un iGPU + une carte dédiée, on garde celle qui a le plus de VRAM.
		if best == nil || g.MemTotal > best.MemTotal {
			best = g
		}
	}
	return best
}

func amdName(dev string) string {
	if b, err := os.ReadFile(filepath.Join(dev, "product_name")); err == nil {
		if n := strings.TrimSpace(string(b)); n != "" {
			return n
		}
	}
	return "AMD Radeon"
}

func readUint(path string) (uint64, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	return v, err == nil
}
