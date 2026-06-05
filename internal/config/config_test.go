package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := Default()
	if d.DailyGoalML != 2000 || d.GlassML != 250 {
		t.Errorf("unexpected volume defaults: %+v", d)
	}
	if d.NotifyMinLevel != "overdue" {
		t.Errorf("notify floor default = %q, want overdue", d.NotifyMinLevel)
	}
	if d.NuclearEscalation {
		t.Error("nuclear escalation should default off")
	}
}

func TestLoadWritesDefaultOnFirstRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg != Default() {
		t.Errorf("first-run config = %+v, want defaults", cfg)
	}
	// The commented template should now exist on disk.
	if _, err := os.Stat(filepath.Join(dir, "hydrate", "config.toml")); err != nil {
		t.Errorf("expected default config written: %v", err)
	}
}

func TestLoadReadsOverrides(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "hydrate"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "daily_goal_ml = 3000\nglass_ml = 500\nunits = \"oz\"\nnuclear_escalation = true\n"
	if err := os.WriteFile(filepath.Join(dir, "hydrate", "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DailyGoalML != 3000 || cfg.GlassML != 500 || cfg.Units != "oz" || !cfg.NuclearEscalation {
		t.Errorf("overrides not applied: %+v", cfg)
	}
	// Unspecified keys fall back to defaults.
	if cfg.DayStartHour != 7 {
		t.Errorf("day_start_hour = %d, want default 7", cfg.DayStartHour)
	}
}
