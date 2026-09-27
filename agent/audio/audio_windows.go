package audio

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Sous Windows on utilise l'API audio native (Core Audio, interfaces COM),
// appelée directement sans cgo :
//   IMMDeviceEnumerator   : sorties, micro, périphériques par défaut
//   IAudioEndpointVolume  : volume et coupure d'un périphérique
//   IAudioSessionManager2 : flux de chaque application (ISimpleAudioVolume)
//   IPolicyConfig         : changer la sortie par défaut (interface non documentée,
//                           utilisée par tous les outils de bascule casque/enceintes)

var (
	ole32                = windows.NewLazySystemDLL("ole32.dll")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procPropVariantClear = ole32.NewProc("PropVariantClear")

	clsidMMDeviceEnumerator = guid("{BCDE0395-E52F-467C-8E3D-C4579291692E}")
	iidIMMDeviceEnumerator  = guid("{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	iidIAudioEndpointVolume = guid("{5CDF2C82-841E-4546-9722-0CF74078229A}")
	iidIAudioSessionManager = guid("{77AA99A0-1BD6-484F-8BC7-2C654C9A9B6F}")
	iidIAudioSessionControl = guid("{BFB7FF88-7239-4FC9-8FA2-07C950BE9C6D}") // IAudioSessionControl2
	iidISimpleAudioVolume   = guid("{87CE5498-68D6-44E5-9215-6DA47EF883D8}")
	clsidPolicyConfig       = guid("{870AF99C-171D-4F9E-AF0D-E63DF40C2BC9}")
	iidIPolicyConfig        = guid("{F8679F50-850A-41CF-9C72-430F290290C8}")

	pkeyFriendlyName = propertyKey{guid("{A45C254E-DF1C-4EFD-8020-67D146A850E0}"), 14}
)

const (
	eRender  = 0
	eCapture = 1

	eConsole        = 0
	eMultimedia     = 1
	eCommunications = 2

	deviceStateActive = 1
	clsctxAll         = 0x17
	vtLPWSTR          = 31
	eNotFound         = 0x80070490

	sessionExpired = 2
)

// Numéros des méthodes dans la table virtuelle de chaque interface.
const (
	mRelease        = 2
	mQueryInterface = 0

	enumEnumAudioEndpoints = 3
	enumGetDefault         = 4

	collGetCount = 3
	collItem     = 4

	devActivate          = 3
	devOpenPropertyStore = 4
	devGetID             = 5

	propGetValue = 5

	epvSetVolume = 7
	epvGetVolume = 9
	epvSetMute   = 14
	epvGetMute   = 15

	smGetSessionEnumerator = 5

	seGetCount   = 3
	seGetSession = 4

	scGetState        = 3
	scGetProcessID    = 14
	scIsSystemSession = 15

	savSetVolume = 3
	savGetVolume = 4
	savSetMute   = 5
	savGetMute   = 6

	pcSetDefaultEndpoint = 13
)

type propertyKey struct {
	fmtid windows.GUID
	pid   uint32
}

// PROPVARIANT (24 octets en 64 bits) : on ne lit que les chaînes.
type propVariant struct {
	vt  uint16
	_   [3]uint16
	str *uint16
	_   uintptr
}

func guid(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}

// com est un pointeur d'interface COM : son premier champ est la table des méthodes.
type com struct{ vtbl *[32]uintptr }

// Comme pour LazyProc.Call, uintptrescapes garde en vie les pointeurs Go passés
// en uintptr (paramètres de sortie) pendant l'appel.
//
//go:uintptrescapes
func (o *com) call(method int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[method], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

func (o *com) release() {
	if o != nil {
		o.call(mRelease)
	}
}

func (o *com) query(iid *windows.GUID) (*com, error) {
	var out *com
	if err := check(o.call(mQueryInterface, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out))), "QueryInterface"); err != nil {
		return nil, err
	}
	return out, nil
}

func check(hr uintptr, what string) error {
	if int32(hr) < 0 {
		return fmt.Errorf("%s : erreur 0x%08X", what, uint32(hr))
	}
	return nil
}

// Les appels COM passent tous par un seul thread système, initialisé une fois.
var (
	comOnce sync.Once
	comJobs = make(chan func())
)

func onCOM(f func() error) error {
	comOnce.Do(func() {
		go func() {
			runtime.LockOSThread()
			if err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED); err != nil {
				log.Printf("audio : CoInitializeEx : %v", err)
			}
			for job := range comJobs {
				job()
			}
		}()
	})
	done := make(chan error, 1)
	comJobs <- func() { done <- f() }
	return <-done
}

func create(clsid, iid *windows.GUID) (*com, error) {
	var out *com
	r, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(clsid)), 0, clsctxAll,
		uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if err := check(r, "CoCreateInstance"); err != nil {
		return nil, err
	}
	return out, nil
}

// withEnumerator ouvre l'énumérateur de périphériques le temps de f.
func withEnumerator(f func(enum *com) error) error {
	return onCOM(func() error {
		enum, err := create(&clsidMMDeviceEnumerator, &iidIMMDeviceEnumerator)
		if err != nil {
			return err
		}
		defer enum.release()
		return f(enum)
	})
}

// defaultDevice renvoie nil (sans erreur) s'il n'y a aucun périphérique de ce type.
func defaultDevice(enum *com, flow, role int) (*com, error) {
	var dev *com
	r := enum.call(enumGetDefault, uintptr(flow), uintptr(role), uintptr(unsafe.Pointer(&dev)))
	if uint32(r) == eNotFound {
		return nil, nil
	}
	if err := check(r, "GetDefaultAudioEndpoint"); err != nil {
		return nil, err
	}
	return dev, nil
}

func activate(dev *com, iid *windows.GUID) (*com, error) {
	var out *com
	if err := check(dev.call(devActivate, uintptr(unsafe.Pointer(iid)), clsctxAll, 0, uintptr(unsafe.Pointer(&out))), "Activate"); err != nil {
		return nil, err
	}
	return out, nil
}

func deviceID(dev *com) string {
	var p *uint16
	if check(dev.call(devGetID, uintptr(unsafe.Pointer(&p))), "GetId") != nil || p == nil {
		return ""
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(p))
	return windows.UTF16PtrToString(p)
}

func friendlyName(dev *com) string {
	var store *com
	if check(dev.call(devOpenPropertyStore, 0 /* STGM_READ */, uintptr(unsafe.Pointer(&store))), "OpenPropertyStore") != nil {
		return ""
	}
	defer store.release()
	var pv propVariant
	if check(store.call(propGetValue, uintptr(unsafe.Pointer(&pkeyFriendlyName)), uintptr(unsafe.Pointer(&pv))), "GetValue") != nil {
		return ""
	}
	defer procPropVariantClear.Call(uintptr(unsafe.Pointer(&pv)))
	if pv.vt != vtLPWSTR || pv.str == nil {
		return ""
	}
	return windows.UTF16PtrToString(pv.str)
}

// Les nombres à virgule sont passés dans les registres XMM : Go y recopie les
// 4 premiers arguments sous Windows 64 bits, il suffit donc de passer les bits.
func f32(v float32) uintptr { return uintptr(math.Float32bits(v)) }

func toPercent(v float32) int { return int(math.Round(float64(v) * 100)) }

func boolArg(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

// endpointLevel lit le volume du périphérique par défaut (nil s'il n'y en a pas).
func endpointLevel(enum *com, flow int) (*Level, error) {
	dev, err := defaultDevice(enum, flow, eConsole)
	if err != nil || dev == nil {
		return nil, err
	}
	defer dev.release()
	epv, err := activate(dev, &iidIAudioEndpointVolume)
	if err != nil {
		return nil, err
	}
	defer epv.release()
	var vol float32
	var muted int32
	if err := check(epv.call(epvGetVolume, uintptr(unsafe.Pointer(&vol))), "GetMasterVolumeLevelScalar"); err != nil {
		return nil, err
	}
	if err := check(epv.call(epvGetMute, uintptr(unsafe.Pointer(&muted))), "GetMute"); err != nil {
		return nil, err
	}
	return &Level{Volume: toPercent(vol), Muted: muted != 0}, nil
}

// withEndpoints applique f au volume des périphériques par défaut de ces rôles
// (sans doublon : souvent le même périphérique a tous les rôles).
func withEndpoints(enum *com, flow int, roles []int, f func(epv *com) error) error {
	seen := map[string]bool{}
	for _, role := range roles {
		dev, err := defaultDevice(enum, flow, role)
		if err != nil {
			return err
		}
		if dev == nil {
			return ErrNotFound
		}
		id := deviceID(dev)
		if seen[id] {
			dev.release()
			continue
		}
		seen[id] = true
		epv, err := activate(dev, &iidIAudioEndpointVolume)
		dev.release()
		if err != nil {
			return err
		}
		err = f(epv)
		epv.release()
		if err != nil {
			return err
		}
	}
	return nil
}

func outputs(enum *com) ([]Device, error) {
	var coll *com
	if err := check(enum.call(enumEnumAudioEndpoints, eRender, deviceStateActive, uintptr(unsafe.Pointer(&coll))), "EnumAudioEndpoints"); err != nil {
		return nil, err
	}
	defer coll.release()

	defID := ""
	if def, err := defaultDevice(enum, eRender, eConsole); err == nil && def != nil {
		defID = deviceID(def)
		def.release()
	}
	var n uint32
	if err := check(coll.call(collGetCount, uintptr(unsafe.Pointer(&n))), "GetCount"); err != nil {
		return nil, err
	}
	var list []Device
	for i := range n {
		var dev *com
		if check(coll.call(collItem, uintptr(i), uintptr(unsafe.Pointer(&dev))), "Item") != nil {
			continue
		}
		id := deviceID(dev)
		name := friendlyName(dev)
		dev.release()
		if name == "" {
			name = "Sortie audio"
		}
		list = append(list, Device{ID: id, Name: name, Default: id != "" && id == defID})
	}
	return list, nil
}

// session est le flux d'une application sur la sortie par défaut.
type session struct {
	exe string
	vol *com // ISimpleAudioVolume
}

// withSessions ouvre les flux des applications sur la sortie par défaut le temps de f.
func withSessions(enum *com, f func([]session) error) error {
	dev, err := defaultDevice(enum, eRender, eConsole)
	if err != nil {
		return err
	}
	if dev == nil {
		return f(nil)
	}
	defer dev.release()
	mgr, err := activate(dev, &iidIAudioSessionManager)
	if err != nil {
		return err
	}
	defer mgr.release()
	var se *com
	if err := check(mgr.call(smGetSessionEnumerator, uintptr(unsafe.Pointer(&se))), "GetSessionEnumerator"); err != nil {
		return err
	}
	defer se.release()
	var n int32
	if err := check(se.call(seGetCount, uintptr(unsafe.Pointer(&n))), "GetCount"); err != nil {
		return err
	}

	var list []session
	defer func() {
		for _, s := range list {
			s.vol.release()
		}
	}()
	for i := range n {
		var ctl *com
		if check(se.call(seGetSession, uintptr(i), uintptr(unsafe.Pointer(&ctl))), "GetSession") != nil {
			continue
		}
		if s, ok := openSession(ctl); ok {
			list = append(list, s)
		}
		ctl.release()
	}
	return f(list)
}

func openSession(ctl *com) (session, bool) {
	ctl2, err := ctl.query(&iidIAudioSessionControl)
	if err != nil {
		return session{}, false
	}
	defer ctl2.release()
	var state int32
	if check(ctl2.call(scGetState, uintptr(unsafe.Pointer(&state))), "GetState") != nil || state == sessionExpired {
		return session{}, false
	}
	if ctl2.call(scIsSystemSession) == 0 { // S_OK : sons système de Windows
		return session{}, false
	}
	var pid uint32
	if check(ctl2.call(scGetProcessID, uintptr(unsafe.Pointer(&pid))), "GetProcessId") != nil || pid == 0 {
		return session{}, false
	}
	exe := exePath(pid)
	if exe == "" {
		return session{}, false
	}
	vol, err := ctl.query(&iidISimpleAudioVolume)
	if err != nil {
		return session{}, false
	}
	return session{exe: exe, vol: vol}, true
}

func exePath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

func sessionID(exe string) string { return strings.ToLower(filepath.Base(exe)) }

func read(context.Context) (State, error) {
	var s State
	err := withEnumerator(func(enum *com) error {
		var err error
		if s.Master, err = endpointLevel(enum, eRender); err != nil {
			return err
		}
		if s.Mic, err = endpointLevel(enum, eCapture); err != nil {
			return err
		}
		if s.Outputs, err = outputs(enum); err != nil {
			return err
		}
		return withSessions(enum, func(list []session) error {
			byID := map[string]int{}
			for _, ses := range list {
				var vol float32
				var muted int32
				if check(ses.vol.call(savGetVolume, uintptr(unsafe.Pointer(&vol))), "GetMasterVolume") != nil ||
					check(ses.vol.call(savGetMute, uintptr(unsafe.Pointer(&muted))), "GetMute") != nil {
					continue
				}
				lvl := Level{Volume: toPercent(vol), Muted: muted != 0}
				id := sessionID(ses.exe)
				if i, ok := byID[id]; ok {
					a := &s.Apps[i]
					a.Volume = max(a.Volume, lvl.Volume)
					a.Muted = a.Muted && lvl.Muted // coupée seulement si tous ses flux le sont
					continue
				}
				byID[id] = len(s.Apps)
				s.Apps = append(s.Apps, App{ID: id, Name: appName(filepath.Base(ses.exe)), Level: lvl, exe: ses.exe})
			}
			return nil
		})
	})
	return s, err
}

// Le micro est coupé sur le périphérique par défaut et sur celui des communications
// (Discord, Teams…) : souvent le même, parfois non.
var micRoles = []int{eConsole, eCommunications}

func setVolume(_ context.Context, t target, volume int) error {
	v := f32(float32(volume) / 100)
	return withEnumerator(func(enum *com) error {
		switch t.kind {
		case "master":
			return withEndpoints(enum, eRender, []int{eConsole}, func(epv *com) error {
				return check(epv.call(epvSetVolume, v, 0), "SetMasterVolumeLevelScalar")
			})
		case "mic":
			return withEndpoints(enum, eCapture, micRoles, func(epv *com) error {
				return check(epv.call(epvSetVolume, v, 0), "SetMasterVolumeLevelScalar")
			})
		}
		return forApp(enum, t.app, func(vol *com) error {
			return check(vol.call(savSetVolume, v, 0), "SetMasterVolume")
		})
	})
}

func setMute(_ context.Context, t target, muted bool) error {
	m := boolArg(muted)
	return withEnumerator(func(enum *com) error {
		switch t.kind {
		case "master":
			return withEndpoints(enum, eRender, []int{eConsole}, func(epv *com) error {
				return check(epv.call(epvSetMute, m, 0), "SetMute")
			})
		case "mic":
			return withEndpoints(enum, eCapture, micRoles, func(epv *com) error {
				return check(epv.call(epvSetMute, m, 0), "SetMute")
			})
		}
		return forApp(enum, t.app, func(vol *com) error {
			return check(vol.call(savSetMute, m, 0), "SetMute")
		})
	})
}

// forApp applique f à tous les flux de l'application.
func forApp(enum *com, id string, f func(vol *com) error) error {
	return withSessions(enum, func(list []session) error {
		found := false
		for _, s := range list {
			if sessionID(s.exe) != id {
				continue
			}
			found = true
			if err := f(s.vol); err != nil {
				return err
			}
		}
		if !found {
			return ErrNotFound
		}
		return nil
	})
}

func setOutput(_ context.Context, id string) error {
	return withEnumerator(func(enum *com) error {
		list, err := outputs(enum)
		if err != nil {
			return err
		}
		found := false
		for _, d := range list {
			found = found || d.ID == id
		}
		if !found {
			return ErrNotFound
		}
		pc, err := create(&clsidPolicyConfig, &iidIPolicyConfig)
		if err != nil {
			return errors.New("changement de sortie indisponible sur cette version de Windows")
		}
		defer pc.release()
		p, err := windows.UTF16PtrFromString(id)
		if err != nil {
			return err
		}
		// Comme le panneau Son de Windows : la sortie devient celle de tous les usages.
		for _, role := range []int{eConsole, eMultimedia, eCommunications} {
			if err := check(pc.call(pcSetDefaultEndpoint, uintptr(unsafe.Pointer(p)), uintptr(role)), "SetDefaultEndpoint"); err != nil {
				return err
			}
		}
		return nil
	})
}
