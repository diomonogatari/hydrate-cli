// Package cli wires hydrate's subcommands together. It is intentionally
// dependency-light: a hand-rolled dispatcher over the standard flag package.
package cli

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime/debug"
	"time"

	"github.com/diomonogatari/hydrate-cli/internal/config"
	"github.com/diomonogatari/hydrate-cli/internal/hydration"
	"github.com/diomonogatari/hydrate-cli/internal/render"
	"github.com/diomonogatari/hydrate-cli/internal/store"
)

// version is set at build time via -ldflags "-X ...cli.version=..." by GoReleaser
// and `make build`. When empty (e.g. a `go install ...@vX.Y.Z` build, which gets
// no ldflags), resolveVersion falls back to the module/VCS info the Go toolchain
// embeds — so released binaries always self-report a real version.
var version = ""

// resolveVersion returns the best available version string for this build.
func resolveVersion() string {
	bi, _ := debug.ReadBuildInfo()
	return versionFrom(version, bi)
}

// versionFrom is the pure resolution logic, separated for testing. Precedence:
// ldflags > module version (`go install module@v…`) > VCS revision > "dev".
func versionFrom(ldflagsVer string, bi *debug.BuildInfo) string {
	if ldflagsVer != "" {
		return ldflagsVer
	}
	if bi == nil {
		return "dev"
	}
	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev, dirty string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if rev != "" {
		if len(rev) > 12 {
			rev = rev[:12]
		}
		return "dev-" + rev + dirty
	}
	return "dev"
}

// Run is the program entry point. It returns a process exit code.
func Run(args []string) int {
	rest := args[1:]

	// Handle help/version first, even in their flag forms (-h/--help/-v/--version),
	// so they aren't swallowed by the default "status" subcommand's flag parser.
	if len(rest) > 0 {
		switch rest[0] {
		case "help", "-h", "--help":
			usage(os.Stdout)
			return 0
		case "version", "-v", "--version":
			fmt.Println("hydrate", resolveVersion())
			return 0
		}
	}

	// Default to `status`; a leading flag (e.g. `hydrate --json`) keeps that default.
	cmd := "status"
	if len(rest) > 0 && !isFlag(rest[0]) {
		cmd, rest = rest[0], rest[1:]
	}

	switch cmd {
	case "init":
		return cmdInit(rest)
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
	case "stats":
		return cmdStats(rest)
	case "config":
		return cmdConfig(rest)
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
// so the bar reacts immediately rather than waiting for the next heartbeat.
func refreshSegment(st hydration.State) {
	pulse := (st.Now.Unix()/pulsePeriodSec)%2 == 1
	seg := render.Segment(st.Level, st.GlassesDone, st.GlassesGoal, pulse)
	_ = store.WriteSegment(seg)
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

func usage(w *os.File) {
	fmt.Fprint(w, `hydrate — a quiet hydration nudge

Usage:
  hydrate init            Interactive setup: profile, heartbeat, shell/tmux wiring
  hydrate [status]        Show today's intake, time since last drink, next due
  hydrate log [AMOUNT]    Log a drink (default one glass; e.g. 500, 500ml, 16oz, 1l)
  hydrate undo            Remove the most recent drink logged today
  hydrate tick            Heartbeat: recompute, refresh the tmux segment (timer-run)
  hydrate segment         Print the styled tmux status string (debug)
  hydrate stats [--days N] History rollup from the log (default last 7 days)
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
