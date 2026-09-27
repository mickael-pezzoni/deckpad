package shortcuts

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

// La capture copie tout l'écran (tous les moniteurs) dans le presse-papiers,
// comme la touche Impr. écran d'avant l'Outil Capture : BitBlt de l'écran vers
// une image, confiée ensuite au presse-papiers.

var (
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	procGetDC              = user32.NewProc("GetDC")
	procReleaseDC          = user32.NewProc("ReleaseDC")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procOpenClipboard      = user32.NewProc("OpenClipboard")
	procEmptyClipboard     = user32.NewProc("EmptyClipboard")
	procSetClipboardData   = user32.NewProc("SetClipboardData")
	procCloseClipboard     = user32.NewProc("CloseClipboard")
	procSetDpiAwareness    = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBM = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
)

const (
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79
	srcCopy           = 0x00CC0020
	captureBlt        = 0x40000000 // inclut les fenêtres transparentes
	cfBitmap          = 2
)

// dpiAware : sans cela, sur un écran à 125 % ou 150 %, Windows donne des tailles
// réduites et seule une partie de l'écran serait capturée.
var dpiAware = sync.OnceFunc(func() {
	if procSetDpiAwareness.Find() == nil {
		const perMonitorV2 = ^uintptr(3) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
		if r, _, _ := procSetDpiAwareness.Call(perMonitorV2); r != 0 {
			return
		}
	}
	procSetProcessDPIAware.Call()
})

func captureAvailable() Availability { return Availability{OK: true} }

func capture() error {
	dpiAware()
	metric := func(i uintptr) int32 {
		r, _, _ := procGetSystemMetrics.Call(i)
		return int32(r)
	}
	x, y := metric(smXVirtualScreen), metric(smYVirtualScreen)
	w, h := metric(smCXVirtualScreen), metric(smCYVirtualScreen)
	if w <= 0 || h <= 0 {
		return fmt.Errorf("taille d'écran inconnue")
	}

	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return fmt.Errorf("écran inaccessible")
	}
	defer procReleaseDC.Call(0, screen)
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	if mem == 0 {
		return fmt.Errorf("capture : CreateCompatibleDC")
	}
	defer procDeleteDC.Call(mem)
	bmp, _, _ := procCreateCompatibleBM.Call(screen, uintptr(w), uintptr(h))
	if bmp == 0 {
		return fmt.Errorf("capture : CreateCompatibleBitmap")
	}
	old, _, _ := procSelectObject.Call(mem, bmp)
	ok, _, err := procBitBlt.Call(mem, 0, 0, uintptr(w), uintptr(h), screen, uintptr(x), uintptr(y), srcCopy|captureBlt)
	procSelectObject.Call(mem, old) // l'image doit être libérée du DC avant d'aller au presse-papiers
	if ok == 0 {
		procDeleteObject.Call(bmp)
		return fmt.Errorf("capture : %v", err)
	}
	if err := toClipboard(bmp); err != nil {
		procDeleteObject.Call(bmp)
		return err
	}
	return nil // le presse-papiers est maintenant propriétaire de l'image
}

func toClipboard(bmp uintptr) error {
	// Une autre appli peut tenir le presse-papiers un court instant.
	var opened uintptr
	for range 10 {
		if opened, _, _ = procOpenClipboard.Call(0); opened != 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if opened == 0 {
		return fmt.Errorf("presse-papiers occupé")
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	if r, _, err := procSetClipboardData.Call(cfBitmap, bmp); r == 0 {
		return fmt.Errorf("presse-papiers : %v", err)
	}
	return nil
}
