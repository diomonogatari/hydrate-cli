// Package config loads and persists hydrate's user settings. On first run it
// writes a commented config.toml with sane defaults; the plaintext file is the
// real artifact and is meant to be edited by hand.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/diomonogatari/hydrate-cli/internal/paths"
)

// Config mirrors config.toml. All values are overridable by the user.
type Config struct {
	DailyGoalML       int    `toml:"daily_goal_ml"`       // target volume per day
	GlassML           int    `toml:"glass_ml"`            // default amount logged by `hydrate log`
	DayStartHour      int    `toml:"day_start_hour"`      // waking window start (local)
	DayEndHour        int    `toml:"day_end_hour"`        // waking window end (local)
	DayResetHour      int    `toml:"day_reset_hour"`      // the "day" rolls over here
	IdleThresholdSec  int    `toml:"idle_threshold_sec"`  // terminal "not in use" after this much shell inactivity
	NotifyMinLevel    string `toml:"notify_min_level"`    // minimum urgency that may trigger a notification
	NotifyCooldownSec int    `toml:"notify_cooldown_sec"` // min gap between notifications
	Units             string `toml:"units"`               // "ml" or "oz" for display only
	NuclearEscalation bool   `toml:"nuclear_escalation"`  // recolor the whole tmux bar at critical
}

// Default returns the shipped configuration. These values must stay in sync
// with defaultTemplate below.
func Default() Config {
	return Config{
		DailyGoalML:       2000,
		GlassML:           250,
		DayStartHour:      7,
		DayEndHour:        23,
		DayResetHour:      4,
		IdleThresholdSec:  600,
		NotifyMinLevel:    "overdue",
		NotifyCooldownSec: 1800,
		Units:             "ml",
		NuclearEscalation: false,
	}
}

// render produces a self-documenting, comment-rich config file from c. It is the
// single template used for both the first-run default and `Save`.
func render(c Config) string {
	return fmt.Sprintf(`# hydrate configuration — https://github.com/diomonogatari/hydrate-cli
# Edit freely; values are read on every command. Re-run `+"`hydrate init`"+` for the wizard.

daily_goal_ml       = %-7d # target volume per day
glass_ml            = %-7d # default amount logged by `+"`hydrate log`"+`
day_start_hour      = %-7d # waking window start (local, 0-23)
day_end_hour        = %-7d # waking window end (local, 0-23)
day_reset_hour      = %-7d # the "day" rolls over here (a 1am glass counts to the prior day)
idle_threshold_sec  = %-7d # terminal considered "not in use" after this much shell inactivity
notify_min_level    = %-9q # minimum urgency that may notify: ok | due | overdue | critical
notify_cooldown_sec = %-7d # minimum gap between desktop notifications (also fires on escalation)
units               = %-9q # "ml" or "oz" — display only
nuclear_escalation  = %-7t # at "critical", recolor the ENTIRE tmux status bar (opt-in, intrusive)
`,
		c.DailyGoalML, c.GlassML, c.DayStartHour, c.DayEndHour, c.DayResetHour,
		c.IdleThresholdSec, c.NotifyMinLevel, c.NotifyCooldownSec, c.Units, c.NuclearEscalation)
}

// Load reads the config file, falling back to (and writing) defaults when it
// does not exist yet. A malformed file surfaces an error rather than silently
// resetting the user's settings.
func Load() (Config, error) {
	cfg := Default()
	path := paths.ConfigFile()

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// Best-effort: write the documented default so the user can discover
		// and tweak it. A write failure is not fatal — defaults still apply.
		_ = writeDefault(path)
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Path returns the resolved config file location.
func Path() string { return paths.ConfigFile() }

// EnsureExists writes the default config if none is present, returning the path.
func EnsureExists() (string, error) {
	path := paths.ConfigFile()
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		if err := writeDefault(path); err != nil {
			return path, err
		}
	}
	return path, nil
}

// Save writes cfg to the config file as a commented TOML document, creating the
// directory if needed. Used by `hydrate init`.
func Save(cfg Config) error {
	return write(paths.ConfigFile(), cfg)
}

func writeDefault(path string) error { return write(path, Default()) }

func write(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(render(cfg)), 0o644)
}
