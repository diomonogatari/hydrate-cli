package notify

import (
	"github.com/godbus/dbus/v5"
)

// Message is a desktop-notification request.
type Message struct {
	Summary    string
	Body       string
	ReplacesID uint32 // 0 = new; otherwise replace the previous notification
	Critical   bool   // urgency=critical; persists until dismissed
}

// Send delivers a notification over the org.freedesktop.Notifications D-Bus
// interface — the same one served by cosmic-notifications, GNOME Shell, KDE,
// dunst, mako, etc. No `notify-send` binary is required.
//
// It returns the server-assigned id, which the caller should persist and pass
// back as ReplacesID so subsequent nudges update the existing notification
// instead of stacking new ones.
func Send(m Message) (uint32, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return 0, err
	}
	// SessionBus returns a shared connection; intentionally not closed.

	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")

	urgency := byte(1)  // normal
	expire := int32(-1) // server default timeout
	if m.Critical {
		urgency = 2 // critical
		expire = 0  // never expire (until dismissed)
	}
	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(urgency),
	}

	call := obj.Call(
		"org.freedesktop.Notifications.Notify", 0,
		"hydrate",    // app_name
		m.ReplacesID, // replaces_id
		"",           // app_icon (the emoji lives in the summary instead)
		m.Summary,    // summary
		m.Body,       // body
		[]string{},   // actions
		hints,        // hints
		expire,       // expire_timeout
	)
	if call.Err != nil {
		return 0, call.Err
	}
	var id uint32
	if err := call.Store(&id); err != nil {
		return 0, err
	}
	return id, nil
}
