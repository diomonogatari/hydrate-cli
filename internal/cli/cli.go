// Package cli wires hydrate's subcommands together. It is intentionally
// dependency-light: a hand-rolled dispatcher over the standard flag package.
package cli

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"

	"github.com/diomonogatari/hydrate-cli/internal/config"
	"github.com/diomonogatari/hydrate-cli/internal/hydration"
	"github.com/diomonogatari/hydrate-cli/internal/render"
	"github.com/diomonogatari/hydrate-cli/internal/store"
)

// version is overridable at build time with -ldflags "-X ...cli.version=...".
var version = "0.1.0-dev"

// Run is the program entry point. It returns a process exit code.
func Run(args []string) int {
	rest := args[1:]

	cmd := "status"
	if len(rest) > 0 && !isFlag(rest[0]) {
		cmd, rest = rest[0], rest[1:]
	}

	switch cmd {
	case "status":
		return cmdStatus(rest)
	case "log":
		return cmdLog(rest)
	case "undo":
		return cmdUndo(rest)
	case "tick":
		return cmdTick(rest)
	case "segment":
		return cmdSegment(rest)
	case "config":
		return cmdConfig(rest)
	case "version", "--version", "-v":
		fmt.Println("hydrate", version)
		return 0
	case "help", "--help", "-h":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "hydrate: unknown command %q\n\n", cmd)
		usage(os.Stderr)
		return 2
	}
}

func isFlag(s string) bool { return len(s) > 0 && s[0] == '-' }

// loadState loads config + log and derives the current hydration state.
func loadState() (config.Config, hydration.State, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, hydration.State{}, fmt.Errorf("reading config: %w", err)
	}
	events, err := store.LoadEvents()
	if err != nil {
		return cfg, hydration.State{}, fmt.Errorf("reading log: %w", err)
	}
	return cfg, hydration.Compute(cfg, events, time.Now()), nil
}

// pulsePeriodSec is the alternation period of the critical "pulse". It matches
// the heartbeat cadence so the segment flips form once per tick.
const pulsePeriodSec = 60

// refreshSegment re-renders the cached tmux string and nudges any running tmux
// so the bar reacts immediately rather than waiting for the next heartbeat. When
// nuclear escalation is enabled it also recolors (or restores) the whole bar.
func refreshSegment(cfg config.Config, st hydration.State) {
	pulse := (st.Now.Unix()/pulsePeriodSec)%2 == 1
	seg := render.Segment(st.Level, st.GlassesDone, st.GlassesGoal, pulse)
	_ = store.WriteSegment(seg)
	if cfg.NuclearEscalation {
		applyNuclear(st.Level == hydration.LevelCritical)
	}
	refreshTmux()
}

// refreshTmux is best-effort: it does nothing when tmux is absent or no server
// is running.
func refreshTmux() {
	if _, err := exec.LookPath("tmux"); err != nil {
		return
	}
	_ = exec.Command("tmux", "refresh-client", "-S").Run()
}

// applyNuclear recolors the entire tmux status bar at critical, and restores the
// default (unsets the override) otherwise. Best-effort; no-op without tmux.
func applyNuclear(on bool) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return
	}
	if on {
		_ = exec.Command("tmux", "set", "-g", "status-style", "bg=colour52").Run()
	} else {
		_ = exec.Command("tmux", "set", "-gu", "status-style").Run()
	}
}

func usage(w *os.File) {
	fmt.Fprint(w, `hydrate — a quiet hydration nudge

Usage:
  hydrate [status]        Show today's intake, time since last drink, next due
  hydrate log [AMOUNT]    Log a drink (default one glass; e.g. 500, 500ml, 16oz, 1l)
  hydrate undo            Remove the most recent drink logged today
  hydrate tick            Heartbeat: recompute, refresh the tmux segment (timer-run)
  hydrate segment         Print the styled tmux status string (debug)
  hydrate config [--edit] Show resolved config and its path; --edit opens $EDITOR
  hydrate version         Print version

Global:
  --json                  Machine-readable output (status, segment)
`)
}

// --- small formatting helpers ---

// progressBar renders a fixed-width [###---] bar for frac in [0,1].
func progressBar(frac float64, width int) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(math.Round(frac * float64(width)))
	bar := make([]byte, width)
	for i := range bar {
		if i < filled {
			bar[i] = '#'
		} else {
			bar[i] = '-'
		}
	}
	return "[" + string(bar) + "]"
}

// displayVolume formats a millilitre amount in the user's configured units.
func displayVolume(ml int, units string) string {
	if units == "oz" {
		return fmt.Sprintf("%.1f oz", float64(ml)/mlPerOz)
	}
	return fmt.Sprintf("%d ml", ml)
}
