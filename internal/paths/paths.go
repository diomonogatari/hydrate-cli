// Package paths resolves the on-disk locations hydrate uses, following the
// XDG Base Directory spec. Everything is derived from XDG_CONFIG_HOME and
// XDG_STATE_HOME with the conventional ~/.config and ~/.local/state fallbacks.
package paths

import (
	"os"
	"path/filepath"
)

// appName is the per-user namespace under the XDG roots.
const appName = "hydrate"

// xdgBase returns the value of envVar, or $HOME/<fallback> when it is unset.
// It deliberately ignores a polluted XDG_DATA_HOME (e.g. inside snaps) by only
// ever consulting the variables hydrate actually relies on: config and state.
func xdgBase(envVar string, fallback ...string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(append([]string{home}, fallback...)...)
}

// ConfigDir is $XDG_CONFIG_HOME/hydrate (default ~/.config/hydrate).
func ConfigDir() string {
	return filepath.Join(xdgBase("XDG_CONFIG_HOME", ".config"), appName)
}

// StateDir is $XDG_STATE_HOME/hydrate (default ~/.local/state/hydrate).
func StateDir() string {
	return filepath.Join(xdgBase("XDG_STATE_HOME", ".local", "state"), appName)
}

// ConfigFile is the user's TOML configuration.
func ConfigFile() string { return filepath.Join(ConfigDir(), "config.toml") }

// LogFile is the append-only JSONL drink log: the single source of truth.
func LogFile() string { return filepath.Join(StateDir(), "log.jsonl") }

// SegmentFile is the pre-rendered tmux string read by the status bar hot path.
func SegmentFile() string { return filepath.Join(StateDir(), "segment") }

// ActivityFile holds the unix timestamp of the last shell activity, stamped by
// the zsh hook. Used (in a later phase) to decide whether the terminal is idle.
func ActivityFile() string { return filepath.Join(StateDir(), "last_activity") }

// NotifyStateFile records the last notification time and level (later phase).
func NotifyStateFile() string { return filepath.Join(StateDir(), "notify_state.json") }
