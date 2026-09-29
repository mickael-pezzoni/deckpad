package desktop

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows refuse qu'un programme en arrière-plan (l'agent) fasse passer une
// fenêtre devant : ce qu'il ouvre reste derrière la fenêtre active. On lève
// donc ce verrou juste avant d'ouvrir, puis on guette la fenêtre qui apparaît
// pour la mettre nous-mêmes au premier plan.

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	shell32                      = windows.NewLazySystemDLL("shell32.dll")
	dwmapi                       = windows.NewLazySystemDLL("dwmapi.dll")
	procShellExecuteEx           = shell32.NewProc("ShellExecuteExW")
	procSendInput                = user32.NewProc("SendInput")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procShowWindow               = user32.NewProc("ShowWindow")
	procIsIconic                 = user32.NewProc("IsIconic")
	procGetWindow                = user32.NewProc("GetWindow")
	procGetWindowLong            = user32.NewProc("GetWindowLongW")
	procGetWindowTextLength      = user32.NewProc("GetWindowTextLengthW")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procDwmGetWindowAttribute    = dwmapi.NewProc("DwmGetWindowAttribute")
)

const (
	seeMaskNoCloseProcess = 0x40
	seeMaskNoAsync        = 0x100
	asfwAny               = ^uintptr(0) // ASFW_ANY (-1)
	swRestore             = 9
	gwOwner               = 4
	gwlExStyle            = ^uintptr(19) // GWL_EXSTYLE (-20)
	wsExToolWindow        = 0x80
	dwmwaCloaked          = 14
	inputMouse            = 0

	watchFor  = 8 * time.Second        // délai max pour voir la fenêtre arriver
	pollEvery = 150 * time.Millisecond // fréquence de la surveillance
	handOff   = time.Second            // laissé à une appli déjà ouverte pour réagir
)

// shellExecuteInfo : structure SHELLEXECUTEINFOW (112 octets en 64 bits).
type shellExecuteInfo struct {
	size          uint32
	mask          uint32
	hwnd          windows.HWND
	verb          *uint16
	file          *uint16
	params        *uint16
	dir           *uint16
	show          int32
	instApp       windows.Handle
	idList        uintptr
	class         *uint16
	keyClass      windows.Handle
	hotKey        uint32
	iconOrMonitor windows.Handle
	process       windows.Handle
}

// mouseInput + remplissage : même taille que la structure INPUT de Windows
// (40 octets en 64 bits, 28 en 32 bits).
type mouseInput struct {
	dx, dy                 int32
	mouseData, flags, time uint32
	extraInfo              uintptr
}

type input struct {
	kind uint32
	mi   mouseInput
}

// shellOpen ouvre la cible comme un double-clic et renvoie le processus lancé
// (0 si l'ouverture est passée par un programme déjà ouvert).
func shellOpen(target string) (windows.Handle, error) {
	verb, _ := windows.UTF16PtrFromString("open")
	file, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return 0, err
	}
	info := shellExecuteInfo{
		mask: seeMaskNoCloseProcess | seeMaskNoAsync,
		verb: verb,
		file: file,
		show: swShowNormal,
	}
	info.size = uint32(unsafe.Sizeof(info))
	if ok, _, err := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&info))); ok == 0 {
		return 0, err
	}
	return info.process, nil
}

// unlockForeground : une entrée factice (souris immobile) redonne à l'agent le
// droit de changer la fenêtre active, qu'il cède aussitôt à tout le monde pour
// que l'appli lancée puisse passer devant d'elle-même.
func unlockForeground() {
	in := input{kind: inputMouse}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
	procAllowSetForegroundWindow.Call(asfwAny)
}

// shownWindows : fenêtres réellement affichées (visibles et pas masquées par
// le bureau virtuel ou les applis du Store en veille).
func shownWindows() map[windows.HWND]bool {
	shown := map[windows.HWND]bool{}
	for _, h := range topWindows() {
		if isShown(h) {
			shown[h] = true
		}
	}
	return shown
}

// raise guette la fenêtre ouverte (nouvelle fenêtre, de préférence celle du
// processus lancé) et la met au premier plan. Si le processus lancé s'arrête
// sans fenêtre, il a passé la main à une appli déjà ouverte (navigateur,
// VS Code…) : on remet devant la fenêtre la plus récente de cette appli.
func raise(before map[windows.HWND]bool, proc windows.Handle) {
	runtime.LockOSThread() // AttachThreadInput est lié au thread
	defer runtime.UnlockOSThread()
	var pid uint32
	var exe string
	if proc != 0 {
		defer windows.CloseHandle(proc)
		pid, _ = windows.GetProcessId(proc)
		exe = imagePath(proc)
	}
	var exitedAt time.Time
	for deadline := time.Now().Add(watchFor); time.Now().Before(deadline); time.Sleep(pollEvery) {
		if h := newWindow(before, pid); h != 0 {
			bringToFront(h)
			return
		}
		if exe == "" {
			continue
		}
		if exitedAt.IsZero() {
			if ev, _ := windows.WaitForSingleObject(proc, 0); ev == windows.WAIT_OBJECT_0 {
				exitedAt = time.Now()
			}
			continue
		}
		if time.Since(exitedAt) >= handOff {
			if h := windowOf(exe); h != 0 {
				bringToFront(h)
			}
			return
		}
	}
}

// newWindow : première fenêtre d'appli apparue depuis la photo « before »,
// dans l'ordre d'empilement, celle du processus lancé en priorité.
func newWindow(before map[windows.HWND]bool, pid uint32) windows.HWND {
	var first windows.HWND
	for _, h := range topWindows() {
		if before[h] || !isAppWindow(h) {
			continue
		}
		if pid == 0 {
			return h
		}
		var owner uint32
		windows.GetWindowThreadProcessId(h, &owner)
		if owner == pid {
			return h
		}
		if first == 0 {
			first = h
		}
	}
	return first
}

// windowOf : fenêtre d'appli la plus haute dont le programme porte le même nom
// que exe (le Bloc-notes du Store tourne ailleurs que notepad.exe qui le lance).
func windowOf(exe string) windows.HWND {
	for _, h := range topWindows() {
		if !isAppWindow(h) {
			continue
		}
		var pid uint32
		windows.GetWindowThreadProcessId(h, &pid)
		p, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
		if err != nil {
			continue
		}
		same := strings.EqualFold(filepath.Base(imagePath(p)), filepath.Base(exe))
		windows.CloseHandle(p)
		if same {
			return h
		}
	}
	return 0
}

// bringToFront met la fenêtre au premier plan (restaurée si réduite). Si
// Windows refuse encore, on s'attache au thread de la fenêtre active : il
// accepte alors le changement.
func bringToFront(h windows.HWND) {
	if r, _, _ := procIsIconic.Call(uintptr(h)); r != 0 {
		procShowWindow.Call(uintptr(h), swRestore)
	}
	if windows.GetForegroundWindow() == h {
		return
	}
	unlockForeground()
	procSetForegroundWindow.Call(uintptr(h))
	fg := windows.GetForegroundWindow()
	if fg == h || fg == 0 {
		return
	}
	fgThread, _ := windows.GetWindowThreadProcessId(fg, nil)
	me := windows.GetCurrentThreadId()
	if fgThread == 0 || fgThread == me {
		return
	}
	if r, _, _ := procAttachThreadInput.Call(uintptr(me), uintptr(fgThread), 1); r == 0 {
		return
	}
	procBringWindowToTop.Call(uintptr(h))
	procSetForegroundWindow.Call(uintptr(h))
	procAttachThreadInput.Call(uintptr(me), uintptr(fgThread), 0)
}

var (
	enumMu  sync.Mutex
	enumOut []windows.HWND
	enumCb  = windows.NewCallback(func(h windows.HWND, _ uintptr) uintptr {
		enumOut = append(enumOut, h)
		return 1
	})
)

// topWindows : fenêtres de premier niveau, de la plus haute à la plus basse.
func topWindows() []windows.HWND {
	enumMu.Lock()
	defer enumMu.Unlock()
	enumOut = nil
	windows.EnumWindows(enumCb, nil)
	return enumOut
}

func isShown(h windows.HWND) bool {
	if !windows.IsWindowVisible(h) {
		return false
	}
	var cloaked uint32
	if procDwmGetWindowAttribute.Find() == nil {
		procDwmGetWindowAttribute.Call(uintptr(h), dwmwaCloaked, uintptr(unsafe.Pointer(&cloaked)), 4)
	}
	return cloaked == 0
}

// isAppWindow : fenêtre affichée qui a sa place dans la barre des tâches
// (pas une infobulle, un menu ou une fenêtre d'outil sans titre).
func isAppWindow(h windows.HWND) bool {
	if !isShown(h) || h == windows.GetShellWindow() {
		return false
	}
	if owner, _, _ := procGetWindow.Call(uintptr(h), gwOwner); owner != 0 {
		return false
	}
	if ex, _, _ := procGetWindowLong.Call(uintptr(h), gwlExStyle); ex&wsExToolWindow != 0 {
		return false
	}
	n, _, _ := procGetWindowTextLength.Call(uintptr(h))
	return n > 0
}

func imagePath(p windows.Handle) string {
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n := uint32(len(buf))
	if windows.QueryFullProcessImageName(p, 0, &buf[0], &n) != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}
