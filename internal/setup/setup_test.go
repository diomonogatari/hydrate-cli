package setup

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestInstallUnitsAndHook(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	assets := fstest.MapFS{
		"packaging/systemd/hydrate.service": {Data: []byte("SERVICE")},
		"packaging/systemd/hydrate.timer":   {Data: []byte("TIMER")},
		"packaging/zsh/hydrate.zsh":         {Data: []byte("HOOK")},
	}

	dir, err := InstallUnits(assets)
	if err != nil {
		t.Fatalf("InstallUnits: %v", err)
	}
	for name, want := range map[string]string{"hydrate.service": "SERVICE", "hydrate.timer": "TIMER"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q (err %v), want %q", name, got, err, want)
		}
	}

	hp, err := InstallHook(assets)
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if got, _ := os.ReadFile(hp); string(got) != "HOOK" {
		t.Errorf("hook = %q, want HOOK", got)
	}
}

func TestInstallUnitsMissingAsset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := InstallUnits(fstest.MapFS{}); err == nil {
		t.Error("expected error when assets are missing")
	}
}
