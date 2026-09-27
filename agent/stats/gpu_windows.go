package stats

import (
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Sous Windows on lit les compteurs de performance (ceux du Gestionnaire des
// tâches) : ils marchent pour toutes les marques mais ne donnent pas la température.
//
// Compteurs utilisés :
//   \GPU Engine(*engtype_3D)\Utilization Percentage  (une instance par processus et par GPU)
//   \GPU Adapter Memory(*)\Dedicated Usage           (une instance par GPU)
// Nom et VRAM totale viennent du registre.

var (
	pdh                              = syscall.NewLazyDLL("pdh.dll")
	procPdhOpenQueryW                = pdh.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounterW        = pdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData          = pdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterArrayW = pdh.NewProc("PdhGetFormattedCounterArrayW")
)

const (
	pdhMoreData    = 0x800007D2
	pdhFmtDouble   = 0x00000200
	pdhFmtNoCap100 = 0x00008000
)

// PDH_FMT_COUNTERVALUE_ITEM_W (64 bits).
type pdhItem struct {
	Name    *uint16
	CStatus uint32
	_       uint32
	Value   float64
}

type gpuCounters struct {
	query, engine, memory uintptr
}

var (
	countersOnce sync.Once
	counters     *gpuCounters // nil si PDH indisponible

	adapterOnce sync.Once
	adapter     struct {
		name string
		vram uint64
	}
)

func readPlatformGPU() *GPU {
	countersOnce.Do(func() { counters = openGPUCounters() })
	if counters == nil {
		return nil
	}
	if r, _, _ := procPdhCollectQueryData.Call(counters.query); r != 0 {
		return nil
	}
	mem, ok := pdhArray(counters.memory)
	if !ok || len(mem) == 0 {
		return nil
	}
	eng, _ := pdhArray(counters.engine) // vide au tout premier appel (compteur de taux)

	// On retient le GPU qui utilise le plus de VRAM : en jeu, c'est la carte dédiée.
	var luid string
	var memUsed float64
	for name, v := range mem {
		if l := luidOf(name); l != "" && v >= memUsed {
			luid, memUsed = l, v
		}
	}
	if luid == "" {
		return nil
	}
	var usage float64
	for name, v := range eng {
		if luidOf(name) == luid {
			usage += v
		}
	}

	adapterOnce.Do(loadAdapterInfo)
	return &GPU{
		Name:     adapter.name,
		Usage:    min(usage, 100),
		MemUsed:  uint64(memUsed),
		MemTotal: adapter.vram,
	}
}

func openGPUCounters() *gpuCounters {
	if pdh.Load() != nil {
		return nil
	}
	var c gpuCounters
	if r, _, _ := procPdhOpenQueryW.Call(0, 0, uintptr(unsafe.Pointer(&c.query))); r != 0 {
		return nil
	}
	add := func(path string, h *uintptr) bool {
		p, _ := syscall.UTF16PtrFromString(path)
		r, _, _ := procPdhAddEnglishCounterW.Call(c.query, uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(h)))
		return r == 0
	}
	if !add(`\GPU Engine(*engtype_3D)\Utilization Percentage`, &c.engine) ||
		!add(`\GPU Adapter Memory(*)\Dedicated Usage`, &c.memory) {
		return nil
	}
	procPdhCollectQueryData.Call(c.query) // première mesure de référence
	return &c
}

// pdhArray renvoie la valeur de chaque instance d'un compteur à joker.
func pdhArray(counter uintptr) (map[string]float64, bool) {
	var size, count uint32
	r, _, _ := procPdhGetFormattedCounterArrayW.Call(counter, pdhFmtDouble|pdhFmtNoCap100,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if r != pdhMoreData || size == 0 {
		return nil, false
	}
	buf := make([]uint64, (size+7)/8) // aligné sur 8 octets
	r, _, _ = procPdhGetFormattedCounterArrayW.Call(counter, pdhFmtDouble|pdhFmtNoCap100,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf[0])))
	if r != 0 {
		return nil, false
	}
	items := unsafe.Slice((*pdhItem)(unsafe.Pointer(&buf[0])), count)
	out := make(map[string]float64, count)
	for _, it := range items {
		if it.CStatus <= 1 { // PDH_CSTATUS_VALID_DATA ou NEW_DATA
			out[windows.UTF16PtrToString(it.Name)] += it.Value
		}
	}
	return out, true
}

// luidOf extrait l'identifiant du GPU d'un nom d'instance,
// ex : "pid_42_luid_0x00000000_0x0000D1B5_phys_0_eng_0_engtype_3D" → "0x00000000_0x0000D1B5".
func luidOf(instance string) string {
	_, rest, ok := strings.Cut(instance, "luid_")
	if !ok {
		return ""
	}
	parts := strings.SplitN(rest, "_", 3)
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "_" + parts[1]
}

// loadAdapterInfo lit le nom et la VRAM de la carte la mieux dotée dans le registre.
func loadAdapterInfo() {
	const class = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, class, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return
	}
	defer root.Close()
	subs, _ := root.ReadSubKeyNames(-1)
	for _, sub := range subs {
		k, err := registry.OpenKey(root, sub, registry.QUERY_VALUE)
		if err != nil {
			continue // ex : « Properties », accès refusé
		}
		name, _, _ := k.GetStringValue("DriverDesc")
		vram, _, err := k.GetIntegerValue("HardwareInformation.qwMemorySize")
		if err != nil {
			vram, _, _ = k.GetIntegerValue("HardwareInformation.MemorySize")
		}
		k.Close()
		if name != "" && vram >= adapter.vram {
			adapter.name, adapter.vram = name, vram
		}
	}
}
