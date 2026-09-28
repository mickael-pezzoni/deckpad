package clipboard

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Sous Windows, on parle directement au presse-papiers (user32) : rien à installer.
// Le numéro de séquence du presse-papiers change à chaque copie : tant qu'il ne
// bouge pas, on ressert le dernier contenu lu sans rien relire ni reconvertir.

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard     = user32.NewProc("OpenClipboard")
	procCloseClipboard    = user32.NewProc("CloseClipboard")
	procEmptyClipboard    = user32.NewProc("EmptyClipboard")
	procGetClipboardData  = user32.NewProc("GetClipboardData")
	procSetClipboardData  = user32.NewProc("SetClipboardData")
	procIsFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")
	procRegisterFormat    = user32.NewProc("RegisterClipboardFormatW")
	procGlobalAlloc       = kernel32.NewProc("GlobalAlloc")
	procGlobalFree        = kernel32.NewProc("GlobalFree")
	procGlobalLock        = kernel32.NewProc("GlobalLock")
	procGlobalUnlock      = kernel32.NewProc("GlobalUnlock")
	procGlobalSize        = kernel32.NewProc("GlobalSize")
	procMoveMemory        = kernel32.NewProc("RtlMoveMemory")
)

const (
	cfUnicodeText = 13
	cfDIB         = 8
	gmemMoveable  = 0x0002
)

// cfPNG : format « PNG » que les navigateurs et beaucoup d'applis ajoutent à côté
// de l'image classique (il garde la transparence, et évite une conversion).
var cfPNG = sync.OnceValue(func() uintptr {
	name, _ := windows.UTF16PtrFromString("PNG")
	r, _, _ := procRegisterFormat.Call(uintptr(unsafe.Pointer(name)))
	return r
})

var (
	lastMu  sync.Mutex
	lastSeq uintptr
	last    content
)

func read(context.Context) (content, error) {
	seq, _, _ := procGetSequenceNumber.Call()
	lastMu.Lock()
	defer lastMu.Unlock()
	if seq != 0 && seq == lastSeq {
		return last, nil
	}
	var c content
	err := withClipboard(func() error {
		c.text = readText()
		c.image = readImage()
		return nil
	})
	if err != nil {
		return content{}, err
	}
	lastSeq, last = seq, c
	return c, nil
}

func write(_ context.Context, text string) error {
	utf16, err := windows.UTF16FromString(text)
	if err != nil {
		return ErrTooLong // caractère nul au milieu du texte
	}
	size := uintptr(len(utf16) * 2)
	h, _, err := procGlobalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return fmt.Errorf("presse-papiers : %v", err)
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		procGlobalFree.Call(h)
		return errors.New("presse-papiers : mémoire inaccessible")
	}
	procMoveMemory.Call(p, uintptr(unsafe.Pointer(&utf16[0])), size)
	procGlobalUnlock.Call(h)

	err = withClipboard(func() error {
		procEmptyClipboard.Call()
		if r, _, err := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
			return fmt.Errorf("presse-papiers : %v", err)
		}
		return nil
	})
	if err != nil {
		procGlobalFree.Call(h)
		return err
	}
	return nil // le presse-papiers est maintenant propriétaire de la mémoire
}

// withClipboard ouvre le presse-papiers le temps de f. Il doit être refermé par
// le même thread : on y reste attaché.
func withClipboard(f func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// Une autre appli peut tenir le presse-papiers un court instant.
	var opened uintptr
	for range 10 {
		if opened, _, _ = procOpenClipboard.Call(0); opened != 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if opened == 0 {
		return errors.New("presse-papiers occupé")
	}
	defer procCloseClipboard.Call()
	return f()
}

// globalBytes lit le bloc de mémoire d'un format du presse-papiers (déjà ouvert).
// Au-delà de limit octets : rien, ou seulement le début si cut.
func globalBytes(format uintptr, limit int, cut bool) []byte {
	if format == 0 {
		return nil
	}
	if ok, _, _ := procIsFormatAvailable.Call(format); ok == 0 {
		return nil
	}
	h, _, _ := procGetClipboardData.Call(format)
	if h == 0 {
		return nil
	}
	size, _, _ := procGlobalSize.Call(h)
	if size > uintptr(limit) {
		if !cut {
			return nil
		}
		size = uintptr(limit)
	}
	if size == 0 {
		return nil
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return nil
	}
	defer procGlobalUnlock.Call(h)
	b := make([]byte, size)
	procMoveMemory.Call(uintptr(unsafe.Pointer(&b[0])), p, size)
	return b
}

func readText() string {
	b := globalBytes(cfUnicodeText, MaxText*2, true)
	if len(b) < 2 {
		return ""
	}
	u := unsafe.Slice((*uint16)(unsafe.Pointer(&b[0])), len(b)/2)
	return windows.UTF16ToString(u) // s'arrête au caractère nul final
}

func readImage() []byte {
	if b := globalBytes(cfPNG(), maxImage, false); isPNG(b) {
		return b
	}
	dib := globalBytes(cfDIB, maxImage, false)
	if dib == nil {
		return nil
	}
	png, err := dibToPNG(dib)
	if err != nil {
		return nil
	}
	return png
}
