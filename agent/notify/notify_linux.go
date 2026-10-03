package notify

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
)

// Sous Linux, le bureau (KDE, GNOME, XFCE…) expose un serveur de notifications
// standard sur le bus de session : org.freedesktop.Notifications.

var (
	busMu sync.Mutex
	bus   *dbus.Conn
)

func session() (*dbus.Conn, error) {
	busMu.Lock()
	defer busMu.Unlock()
	if bus != nil && bus.Connected() {
		return bus, nil
	}
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("bus de session D-Bus inaccessible : %w", err)
	}
	bus = c
	return c, nil
}

func showSystem(m Message) error {
	conn, err := session()
	if err != nil {
		return err
	}
	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(byte(0)), // basse : simple information
	}
	call := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").Call(
		"org.freedesktop.Notifications.Notify", 0,
		"deckpad",  // nom de l'application
		uint32(0),  // nouvelle notification
		iconPath(), // icône (chemin du fichier)
		m.Title,
		m.Body,
		[]string{}, // pas de boutons
		hints,
		int32(-1), // durée par défaut du bureau
	)
	return call.Err
}
