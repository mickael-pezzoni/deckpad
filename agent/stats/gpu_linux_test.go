package stats

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadAMDSysfs(t *testing.T) {
	drm := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(drm, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content+"\n"), 0o644)
	}
	// card0 : iGPU (petite VRAM), card1 : carte dédiée.
	write("card0/device/gpu_busy_percent", "3")
	write("card0/device/mem_info_vram_total", "536870912")
	write("card1/device/gpu_busy_percent", "72")
	write("card1/device/mem_info_vram_used", "4294967296")
	write("card1/device/mem_info_vram_total", "17179869184")
	write("card1/device/hwmon/hwmon3/temp1_input", "61000")
	write("card1-DP-1/status", "connected") // sortie écran, à ignorer

	g := readAMDSysfs(drm)
	if g == nil {
		t.Fatal("GPU attendu")
	}
	if g.Usage != 72 || g.MemUsed != 4<<30 || g.MemTotal != 16<<30 {
		t.Errorf("mauvaise carte ou lecture : %+v", g)
	}
	if g.Temp == nil || *g.Temp != 61 {
		t.Errorf("température : %v", g.Temp)
	}
	if readAMDSysfs(t.TempDir()) != nil {
		t.Error("sans carte AMD, nil attendu")
	}
}
