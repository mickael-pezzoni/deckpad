package notify

import (
	"encoding/xml"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Sous Windows, on affiche un toast (centre de notifications) par l'API WinRT
// Windows.UI.Notifications, appelée directement sans cgo. Un programme non
// installé par le Store se déclare d'abord dans le registre de l'utilisateur
// (HKCU\Software\Classes\AppUserModelId\deckpad) : c'est là que Windows lit le
// nom et l'icône affichés en tête de la notification.

const appID = "deckpad"

var (
	combase                    = windows.NewLazySystemDLL("combase.dll")
	procRoInitialize           = combase.NewProc("RoInitialize")
	procRoActivateInstance     = combase.NewProc("RoActivateInstance")
	procRoGetActivationFactory = combase.NewProc("RoGetActivationFactory")
	procWindowsCreateString    = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString    = combase.NewProc("WindowsDeleteString")

	iidXmlDocument         = guid("{F7F3A506-1E87-42D6-BCFB-B8C809FA5494}")
	iidXmlDocumentIO       = guid("{6CD0E74E-EE65-4489-9EBF-CA43E87BA637}")
	iidToastFactory        = guid("{04124B20-82C6-4229-B109-FD9ED4662B53}")
	iidToastManagerStatics = guid("{50AC103F-D235-4598-BBEF-98FE4D1A3AD4}")
)

// Numéros des méthodes dans la table virtuelle (6 = première méthode après IInspectable).
const (
	mQueryInterface = 0
	mRelease        = 2

	xmlLoadXml                  = 6 // IXmlDocumentIO
	factoryCreateToast          = 6 // IToastNotificationFactory
	managerCreateNotifierWithID = 7 // IToastNotificationManagerStatics
	notifierShow                = 6 // IToastNotifier
)

func guid(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}

// obj est un pointeur d'interface COM/WinRT : son premier champ est la table des méthodes.
type obj struct{ vtbl *[16]uintptr }

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

func (o *obj) query(iid *windows.GUID) (*obj, error) {
	var out *obj
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

// hstring crée une chaîne WinRT, à libérer avec WindowsDeleteString.
func hstring(s string) (uintptr, error) {
	if s == "" {
		return 0, nil // la chaîne vide de WinRT
	}
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return 0, err
	}
	var h uintptr
	r, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&h)))
	return h, check(r, "WindowsCreateString")
}

func factory(class string, iid *windows.GUID) (*obj, error) {
	h, err := hstring(class)
	if err != nil {
		return nil, err
	}
	defer procWindowsDeleteString.Call(h)
	var f *obj
	r, _, _ := procRoGetActivationFactory.Call(h, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&f)))
	return f, check(r, class)
}

// Le toast est construit et affiché sur un thread système dédié, initialisé une fois.
var (
	rtOnce sync.Once
	rtErr  error
	rtJobs = make(chan func())
	regErr error
)

func onWinRT(f func() error) error {
	rtOnce.Do(func() {
		ready := make(chan struct{})
		go func() {
			runtime.LockOSThread()
			if r, _, _ := procRoInitialize.Call(1 /* RO_INIT_MULTITHREADED */); int32(r) < 0 {
				rtErr = check(r, "RoInitialize")
			}
			regErr = register()
			close(ready)
			for job := range rtJobs {
				job()
			}
		}()
		<-ready
	})
	if rtErr != nil {
		return rtErr
	}
	done := make(chan error, 1)
	rtJobs <- func() { done <- f() }
	return <-done
}

// register déclare deckpad (nom et icône) dans le registre de l'utilisateur.
func register() error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\AppUserModelId\`+appID, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("registre : %w", err)
	}
	defer k.Close()
	if err := k.SetStringValue("DisplayName", "deckpad"); err != nil {
		return fmt.Errorf("registre : %w", err)
	}
	if p := iconPath(); p != "" {
		k.SetStringValue("IconUri", p)
	}
	return nil
}

// toastXML : le modèle de notification générique, un titre et une ligne.
func toastXML(m Message) string {
	esc := func(s string) string {
		var b strings.Builder
		xml.EscapeText(&b, []byte(s))
		return b.String()
	}
	var b strings.Builder
	b.WriteString(`<toast><visual><binding template="ToastGeneric"><text>`)
	b.WriteString(esc(m.Title))
	b.WriteString(`</text>`)
	if m.Body != "" {
		b.WriteString(`<text>` + esc(m.Body) + `</text>`)
	}
	b.WriteString(`</binding></visual></toast>`)
	return b.String()
}

func showSystem(m Message) error {
	return onWinRT(func() error {
		if regErr != nil {
			return regErr
		}
		// Le contenu : un document XML WinRT chargé depuis le texte.
		cls, err := hstring("Windows.Data.Xml.Dom.XmlDocument")
		if err != nil {
			return err
		}
		var inst *obj
		r, _, _ := procRoActivateInstance.Call(cls, uintptr(unsafe.Pointer(&inst)))
		procWindowsDeleteString.Call(cls)
		if err := check(r, "XmlDocument"); err != nil {
			return fmt.Errorf("notifications Windows indisponibles : %w", err)
		}
		defer inst.release()
		io, err := inst.query(&iidXmlDocumentIO)
		if err != nil {
			return err
		}
		defer io.release()
		text, err := hstring(toastXML(m))
		if err != nil {
			return err
		}
		r = io.call(xmlLoadXml, text)
		procWindowsDeleteString.Call(text)
		if err := check(r, "LoadXml"); err != nil {
			return err
		}
		doc, err := inst.query(&iidXmlDocument)
		if err != nil {
			return err
		}
		defer doc.release()

		// La notification, puis l'émetteur au nom de deckpad.
		tf, err := factory("Windows.UI.Notifications.ToastNotification", &iidToastFactory)
		if err != nil {
			return err
		}
		defer tf.release()
		var toast *obj
		if err := check(tf.call(factoryCreateToast, uintptr(unsafe.Pointer(doc)), uintptr(unsafe.Pointer(&toast))), "CreateToastNotification"); err != nil {
			return err
		}
		defer toast.release()

		mgr, err := factory("Windows.UI.Notifications.ToastNotificationManager", &iidToastManagerStatics)
		if err != nil {
			return err
		}
		defer mgr.release()
		id, err := hstring(appID)
		if err != nil {
			return err
		}
		var notifier *obj
		r = mgr.call(managerCreateNotifierWithID, id, uintptr(unsafe.Pointer(&notifier)))
		procWindowsDeleteString.Call(id)
		if err := check(r, "CreateToastNotifier"); err != nil {
			return err
		}
		defer notifier.release()
		return check(notifier.call(notifierShow, uintptr(unsafe.Pointer(toast))), "Show")
	})
}
