package media

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Sous Windows, les lecteurs (Spotify, navigateurs, lecteur multimédia…) publient
// leur lecture auprès du système : c'est le panneau média qui s'affiche avec les
// touches de volume. On lit ces « sessions média » par l'API WinRT
// Windows.Media.Control (Windows 10 1809 et plus), appelée directement sans cgo :
//   GlobalSystemMediaTransportControlsSessionManager : session en cours
//   ...Session                  : titre, état, commandes (lecture, pause, suivant…)
//   IRandomAccessStreamReference : pochette, lue comme un IStream classique

var (
	combase                       = windows.NewLazySystemDLL("combase.dll")
	procRoInitialize              = combase.NewProc("RoInitialize")
	procRoGetActivationFactory    = combase.NewProc("RoGetActivationFactory")
	procWindowsCreateString       = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString       = combase.NewProc("WindowsDeleteString")
	procWindowsGetStringRawBuffer = combase.NewProc("WindowsGetStringRawBuffer")

	shcore                                 = windows.NewLazySystemDLL("shcore.dll")
	procCreateStreamOverRandomAccessStream = shcore.NewProc("CreateStreamOverRandomAccessStream")

	iidManagerStatics = guid("{2050C4EE-11A0-57DE-AED7-C97C70338245}")
	iidIAsyncInfo     = guid("{00000036-0000-0000-C000-000000000046}")
	iidIStream        = guid("{0000000C-0000-0000-C000-000000000046}")
)

const managerClass = "Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager"

// Numéros des méthodes dans la table virtuelle (6 = première méthode après IInspectable).
const (
	mQueryInterface = 0
	mRelease        = 2

	staticsRequestAsync = 6

	managerGetCurrentSession = 6

	sessionSourceAppUserModelID  = 6
	sessionTryGetMediaProperties = 7
	sessionGetPlaybackInfo       = 9
	sessionTryPlay               = 10
	sessionTryPause              = 11
	sessionTrySkipNext           = 16
	sessionTrySkipPrevious       = 17

	propsTitle      = 6
	propsArtist     = 9
	propsAlbumTitle = 10
	propsThumbnail  = 15

	playbackStatus = 7

	asyncGetResults = 8 // IAsyncOperation<T>
	asyncGetStatus  = 7 // IAsyncInfo

	streamRefOpenRead = 6

	istreamRead = 3
)

// États d'une opération asynchrone et d'une lecture.
const (
	asyncStarted   = 0
	asyncCompleted = 1

	statusPlaying = 4
)

func guid(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}

// obj est un pointeur d'interface COM/WinRT : son premier champ est la table des méthodes.
type obj struct{ vtbl *[32]uintptr }

//go:uintptrescapes
func (o *obj) call(method int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[method], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

func (o *obj) release() {
	if o != nil {
		o.call(mRelease)
	}
}

// get appelle une méthode qui renvoie une interface (nil sans erreur si elle n'existe pas).
func (o *obj) get(method int, what string) (*obj, error) {
	var out *obj
	if err := check(o.call(method, uintptr(unsafe.Pointer(&out))), what); err != nil {
		return nil, err
	}
	return out, nil
}

func (o *obj) query(iid *windows.GUID) (*obj, error) {
	var out *obj
	if err := check(o.call(mQueryInterface, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out))), "QueryInterface"); err != nil {
		return nil, err
	}
	return out, nil
}

// str lit une propriété texte (HSTRING).
func (o *obj) str(method int) string {
	var h uintptr
	if check(o.call(method, uintptr(unsafe.Pointer(&h))), "HSTRING") != nil || h == 0 {
		return ""
	}
	defer procWindowsDeleteString.Call(h)
	var n uint32
	p, _, _ := procWindowsGetStringRawBuffer.Call(h, uintptr(unsafe.Pointer(&n)))
	if p == 0 || n == 0 {
		return ""
	}
	buf := *(**uint16)(unsafe.Pointer(&p)) // mémoire gérée par Windows, pas par Go
	return strings.TrimSpace(windows.UTF16ToString(unsafe.Slice(buf, n)))
}

func check(hr uintptr, what string) error {
	if int32(hr) < 0 {
		return fmt.Errorf("%s : erreur 0x%08X", what, uint32(hr))
	}
	return nil
}

// await attend la fin d'une opération WinRT (IAsyncOperation<T>) puis renvoie son résultat.
// On interroge son état plutôt que d'implémenter un objet de rappel COM.
func await(op *obj, result uintptr, what string) error {
	info, err := op.query(&iidIAsyncInfo)
	if err != nil {
		return err
	}
	defer info.release()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var status int32
		if err := check(info.call(asyncGetStatus, uintptr(unsafe.Pointer(&status))), what); err != nil {
			return err
		}
		if status == asyncCompleted {
			break
		}
		if status != asyncStarted {
			return fmt.Errorf("%s : échec", what)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s : pas de réponse", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
	return check(op.call(asyncGetResults, result), what)
}

// Les appels WinRT passent tous par un seul thread système, initialisé une fois.
var (
	rtOnce  sync.Once
	rtJobs  = make(chan func())
	manager *obj // gestionnaire des sessions, gardé ouvert
)

func onWinRT(f func() error) error {
	rtOnce.Do(func() {
		go func() {
			runtime.LockOSThread()
			if r, _, _ := procRoInitialize.Call(1 /* RO_INIT_MULTITHREADED */); int32(r) < 0 {
				err := check(r, "RoInitialize")
				log.Printf("médias : RoInitialize : %v", err)
			}
			for job := range rtJobs {
				job()
			}
		}()
	})
	done := make(chan error, 1)
	rtJobs <- func() { done <- f() }
	return <-done
}

func openManager() (*obj, error) {
	if manager != nil {
		return manager, nil
	}
	name, err := windows.UTF16FromString(managerClass)
	if err != nil {
		return nil, err
	}
	var h uintptr
	r, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)-1), uintptr(unsafe.Pointer(&h)))
	if err := check(r, "WindowsCreateString"); err != nil {
		return nil, err
	}
	defer procWindowsDeleteString.Call(h)
	var statics *obj
	r, _, _ = procRoGetActivationFactory.Call(h, uintptr(unsafe.Pointer(&iidManagerStatics)), uintptr(unsafe.Pointer(&statics)))
	if err := check(r, "RoGetActivationFactory"); err != nil {
		return nil, fmt.Errorf("commandes média Windows indisponibles (Windows 10 1809 minimum) : %w", err)
	}
	defer statics.release()
	op, err := statics.get(staticsRequestAsync, "RequestAsync")
	if err != nil {
		return nil, err
	}
	defer op.release()
	var m *obj
	if err := await(op, uintptr(unsafe.Pointer(&m)), "RequestAsync"); err != nil {
		return nil, err
	}
	manager = m
	return m, nil
}

// withSession donne à f la session média en cours (nil s'il n'y en a pas).
func withSession(f func(s *obj) error) error {
	return onWinRT(func() error {
		m, err := openManager()
		if err != nil {
			return err
		}
		s, err := m.get(managerGetCurrentSession, "GetCurrentSession")
		if err != nil {
			return err
		}
		if s != nil {
			defer s.release()
		}
		return f(s)
	})
}

func mediaProperties(s *obj) (*obj, error) {
	op, err := s.get(sessionTryGetMediaProperties, "TryGetMediaPropertiesAsync")
	if err != nil {
		return nil, err
	}
	defer op.release()
	var props *obj
	if err := await(op, uintptr(unsafe.Pointer(&props)), "TryGetMediaPropertiesAsync"); err != nil {
		return nil, err
	}
	if props == nil {
		return nil, ErrNoPlayer
	}
	return props, nil
}

func isPlaying(s *obj) bool {
	info, err := s.get(sessionGetPlaybackInfo, "GetPlaybackInfo")
	if err != nil || info == nil {
		return false
	}
	defer info.release()
	var status int32
	return check(info.call(playbackStatus, uintptr(unsafe.Pointer(&status))), "PlaybackStatus") == nil && status == statusPlaying
}

func read(context.Context) (State, error) {
	var st State
	err := withSession(func(s *obj) error {
		if s == nil {
			return nil
		}
		st.Active = true
		st.Playing = isPlaying(s)
		st.App = sourceApp(s.str(sessionSourceAppUserModelID))
		props, err := mediaProperties(s)
		if err != nil {
			return nil // lecteur ouvert mais sans informations sur le morceau
		}
		defer props.release()
		st.Title = props.str(propsTitle)
		st.Artist = props.str(propsArtist)
		st.Album = props.str(propsAlbumTitle)
		if thumb, err := props.get(propsThumbnail, "Thumbnail"); err == nil && thumb != nil {
			thumb.release()
			// La pochette se lit à la demande : on la repère par le morceau.
			st.coverRef = st.App + "\x00" + st.Title + "\x00" + st.Artist + "\x00" + st.Album
		}
		return nil
	})
	return st, err
}

func control(_ context.Context, a Action) error {
	return withSession(func(s *obj) error {
		if s == nil {
			return ErrNoPlayer
		}
		method := map[Action]int{Next: sessionTrySkipNext, Previous: sessionTrySkipPrevious}[a]
		if a == PlayPause {
			method = sessionTryPlay
			if isPlaying(s) {
				method = sessionTryPause
			}
		}
		op, err := s.get(method, "commande")
		if err != nil {
			return err
		}
		defer op.release()
		var ok uint8
		if err := await(op, uintptr(unsafe.Pointer(&ok)), "commande"); err != nil {
			return err
		}
		if ok == 0 {
			return ErrRefused
		}
		return nil
	})
}

// readCover lit la pochette de la session en cours.
func readCover(context.Context, string) ([]byte, error) {
	var data []byte
	err := withSession(func(s *obj) error {
		if s == nil {
			return ErrNoCover
		}
		props, err := mediaProperties(s)
		if err != nil {
			return ErrNoCover
		}
		defer props.release()
		thumb, err := props.get(propsThumbnail, "Thumbnail")
		if err != nil || thumb == nil {
			return ErrNoCover
		}
		defer thumb.release()
		op, err := thumb.get(streamRefOpenRead, "OpenReadAsync")
		if err != nil {
			return err
		}
		defer op.release()
		var ras *obj
		if err := await(op, uintptr(unsafe.Pointer(&ras)), "OpenReadAsync"); err != nil {
			return err
		}
		if ras == nil {
			return ErrNoCover
		}
		defer ras.release()
		var stream *obj
		r, _, _ := procCreateStreamOverRandomAccessStream.Call(uintptr(unsafe.Pointer(ras)), uintptr(unsafe.Pointer(&iidIStream)), uintptr(unsafe.Pointer(&stream)))
		if err := check(r, "CreateStreamOverRandomAccessStream"); err != nil {
			return err
		}
		defer stream.release()
		buf := make([]byte, 64<<10)
		for len(data) < maxCover {
			var n uint32
			hr := stream.call(istreamRead, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&n)))
			if err := check(hr, "Read"); err != nil {
				return err
			}
			data = append(data, buf[:n]...)
			if n == 0 || hr != 0 { // S_FALSE : fin du flux
				break
			}
		}
		if len(data) == 0 {
			return ErrNoCover
		}
		return nil
	})
	return data, err
}
