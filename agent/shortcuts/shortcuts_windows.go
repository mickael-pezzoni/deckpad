package shortcuts

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32            = windows.NewLazySystemDLL("user32.dll")
	procSendInput     = user32.NewProc("SendInput")
	procMapVirtualKey = user32.NewProc("MapVirtualKeyW")
)

const (
	inputKeyboard    = 1
	keyEventExtended = 0x0001
	keyEventKeyUp    = 0x0002
	mapVKToScanCode  = 0
	createNoWindow   = 0x08000000
	swShowNormal     = 1
)

// keyboardInput + remplissage : même taille que la structure INPUT de Windows
// (40 octets en 64 bits, 28 en 32 bits).
type keyboardInput struct {
	vk, scan  uint16
	flags     uint32
	time      uint32
	extraInfo uintptr
}

type input struct {
	kind uint32
	ki   keyboardInput
	_    [8]byte
}

type vkey struct {
	code     uint16
	extended bool // touches du pavé de navigation (flèches, Inser, Suppr…)
}

var vkeys = func() map[string]vkey {
	m := map[string]vkey{
		"ctrl": {0x11, false}, "alt": {0x12, false}, "shift": {0x10, false}, "win": {0x5B, true},
		"esc": {0x1B, false}, "enter": {0x0D, false}, "tab": {0x09, false}, "space": {0x20, false},
		"backspace": {0x08, false}, "delete": {0x2E, true}, "insert": {0x2D, true},
		"printscreen": {0x2C, true}, "up": {0x26, true}, "down": {0x28, true},
		"left": {0x25, true}, "right": {0x27, true}, "home": {0x24, true}, "end": {0x23, true},
		"pageup": {0x21, true}, "pagedown": {0x22, true},
	}
	for c := 'a'; c <= 'z'; c++ {
		m[string(c)] = vkey{uint16(c - 'a' + 'A'), false}
	}
	for c := '0'; c <= '9'; c++ {
		m[string(c)] = vkey{uint16(c), false}
	}
	for i := 1; i <= 12; i++ {
		m[fmt.Sprintf("f%d", i)] = vkey{uint16(0x70 + i - 1), false}
	}
	return m
}()

func keysAvailable() Availability { return Availability{OK: true} }

// sendKeys appuie sur les touches dans l'ordre puis les relâche dans l'ordre inverse,
// comme si l'utilisateur les tapait (SendInput).
func sendKeys(keys []string) error {
	var in []input
	for _, k := range keys {
		in = append(in, keyEvent(k, false))
	}
	for i := len(keys) - 1; i >= 0; i-- {
		in = append(in, keyEvent(keys[i], true))
	}
	n, _, err := procSendInput.Call(uintptr(len(in)), uintptr(unsafe.Pointer(&in[0])), unsafe.Sizeof(in[0]))
	if int(n) != len(in) {
		return fmt.Errorf("envoi des touches : %v", err)
	}
	return nil
}

func keyEvent(name string, up bool) input {
	k := vkeys[name]
	scan, _, _ := procMapVirtualKey.Call(uintptr(k.code), mapVKToScanCode)
	var flags uint32
	if k.extended {
		flags |= keyEventExtended
	}
	if up {
		flags |= keyEventKeyUp
	}
	return input{kind: inputKeyboard, ki: keyboardInput{vk: k.code, scan: uint16(scan), flags: flags}}
}

// launch passe par « start » pour que les programmes graphiques s'ouvrent
// normalement, sans fenêtre de console.
func launch(command string) error {
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
		CmdLine:       `cmd /d /c start "" ` + command,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// open ouvre un dossier, un fichier ou une adresse avec le programme associé.
func open(target string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	file, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, nil, swShowNormal)
}

func defaults() []Shortcut {
	return []Shortcut{
		{ID: "capture", Label: "Capture d'écran", Icon: "camera", Color: "blue", Kind: KindCapture},
		{ID: "record", Label: "Enregistrer l'écran", Icon: "video", Color: "red", Kind: KindKeys, Keys: []string{"alt", "win", "r"}},
		{ID: "taskmgr", Label: "Gestionnaire des tâches", Icon: "gauge", Color: "green", Kind: KindKeys, Keys: []string{"ctrl", "shift", "esc"}},
		{ID: "home", Label: "Dossier personnel", Icon: "folder", Color: "orange", Kind: KindOpen, Target: "~"},
		{ID: "desktop", Label: "Afficher le bureau", Icon: "monitor", Color: "purple", Kind: KindKeys, Keys: []string{"win", "d"}},
		{ID: "calc", Label: "Calculatrice", Icon: "calculator", Color: "teal", Kind: KindLaunch, Command: "calc"},
	}
}

const keyEventUnicode = 0x0004

func typeAvailable() Availability { return Availability{OK: true} }

// typeText envoie chaque caractère tel quel (KEYEVENTF_UNICODE) : il s'affiche
// comme prévu quelle que soit la disposition du clavier. Retour à la ligne et
// tabulation passent par les vraies touches, que les applis attendent.
func typeText(text string) error {
	var in []input
	for _, r := range text {
		switch r {
		case '\n':
			in = append(in, keyEvent("enter", false), keyEvent("enter", true))
		case '\t':
			in = append(in, keyEvent("tab", false), keyEvent("tab", true))
		default:
			for _, u := range utf16.Encode([]rune{r}) {
				in = append(in,
					input{kind: inputKeyboard, ki: keyboardInput{scan: u, flags: keyEventUnicode}},
					input{kind: inputKeyboard, ki: keyboardInput{scan: u, flags: keyEventUnicode | keyEventKeyUp}})
			}
		}
	}
	// Par paquets : certaines applis perdent des caractères s'ils arrivent tous d'un coup.
	for len(in) > 0 {
		chunk := in[:min(len(in), 64)]
		in = in[len(chunk):]
		n, _, err := procSendInput.Call(uintptr(len(chunk)), uintptr(unsafe.Pointer(&chunk[0])), unsafe.Sizeof(chunk[0]))
		if int(n) != len(chunk) {
			return fmt.Errorf("saisie du texte : %v", err)
		}
		if len(in) > 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	return nil
}
