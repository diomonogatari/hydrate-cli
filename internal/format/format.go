// Package format holds small, dependency-free display helpers shared across
// the CLI and the notifier.
package format

import (
	"fmt"
	"time"
)

// HumanizeDuration renders a duration as "1h 40m" / "5m" / "just now".
func HumanizeDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
