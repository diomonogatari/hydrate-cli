// Package render turns a hydration Level into the styled tmux string cached for
// the status-bar hot path. The status bar only ever cats the cache file — it
// never invokes the binary — so all styling decisions live here.
package render

import (
	"fmt"

	"github.com/diomonogatari/hydrate-cli/internal/hydration"
)

// Segment returns a fully-styled tmux string for the given level and progress.
// Lower levels stay subtle; critical is built to catch peripheral vision with a
// bright background, bold, and blink.
func Segment(level hydration.Level, done, goal int) string {
	count := fmt.Sprintf("%d/%d", done, goal)
	switch level {
	case hydration.LevelOK:
		return fmt.Sprintf("#[fg=blue]💧 %s#[default]", count)
	case hydration.LevelDue:
		return fmt.Sprintf("#[fg=yellow]💧 %s#[default]", count)
	case hydration.LevelOverdue:
		return fmt.Sprintf("#[fg=colour208,bold]💧 %s ·due·#[default]", count)
	case hydration.LevelCritical:
		return fmt.Sprintf("#[bg=colour196,fg=white,bold,blink] 💧 DRINK WATER %s #[default]", count)
	default:
		return fmt.Sprintf("#[fg=blue]💧 %s#[default]", count)
	}
}
