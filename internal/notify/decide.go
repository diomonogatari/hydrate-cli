// Package notify decides whether a desktop notification should fire and renders
// its content. The decision (Decide) is a pure function so the focus/idle/
// cooldown policy can be exhaustively unit-tested; the actual delivery lives in
// send.go behind the freedesktop D-Bus interface.
package notify

import (
	"fmt"

	"github.com/diomonogatari/hydrate-cli/internal/format"
	"github.com/diomonogatari/hydrate-cli/internal/hydration"
)

// Inputs are everything the policy needs, gathered by the caller. Keeping this a
// plain value (no I/O) is what makes Decide testable.
type Inputs struct {
	Level            hydration.Level // current urgency
	MinLevel         hydration.Level // config notify floor
	Now              int64           // unix seconds
	LastActivity     int64           // unix seconds of last shell activity
	HaveActivity     bool            // whether an activity timestamp exists
	IdleThresholdSec int             // "terminal in use" window
	Focused          *bool           // optional focus probe: nil = unknown
	PrevLevel        hydration.Level // level at the last notification
	PrevNotifyTS     int64           // unix seconds of the last notification
	CooldownSec      int             // minimum gap between notifications
}

// Decision is the verdict plus a short human-readable reason (handy for logs and
// the --json tick output).
type Decision struct {
	Send   bool
	Reason string
}

// Decide implements the suppression policy from the design:
//   - never below the configured floor;
//   - never while the terminal is actively in use (reliable primary gate);
//   - never while the terminal is the focused window (optional, when known);
//   - otherwise fire on escalation, or once the cooldown has elapsed.
func Decide(in Inputs) Decision {
	if in.Level < in.MinLevel {
		return Decision{false, "below notify floor"}
	}
	// A missing activity stamp means "unknown" — treat as not-in-use so the tool
	// still reaches a user who hasn't installed the shell hook. With the hook,
	// active typing reliably suppresses notifications.
	if in.HaveActivity && in.Now-in.LastActivity < int64(in.IdleThresholdSec) {
		return Decision{false, "terminal in use"}
	}
	if in.Focused != nil && *in.Focused {
		return Decision{false, "terminal focused"}
	}
	if in.Level > in.PrevLevel {
		return Decision{true, "escalated"}
	}
	if in.Now-in.PrevNotifyTS >= int64(in.CooldownSec) {
		return Decision{true, "cooldown elapsed"}
	}
	return Decision{false, "within cooldown"}
}

// Compose builds the notification title and body from the current state.
func Compose(st hydration.State) (summary, body string) {
	summary = "Drink water 💧"
	body = fmt.Sprintf("It's been %s — %d/%d glasses today.",
		format.HumanizeDuration(st.SinceLast), st.GlassesDone, st.GlassesGoal)
	return summary, body
}
