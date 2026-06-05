package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestInstallUnitsAndHook(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	assets := fstest.MapFS{
		"systemd/hydrate.service": {Data: []byte("SERVICE")},
		"systemd/hydrate.timer":   {Data: []byte("TIMER")},
		"zsh/hydrate.zsh":         {Data: []byte("HOOK")},
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

func TestWireZshIdempotentWithBackup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZDOTDIR", dir)
	rc := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(rc, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// First wire: changes the file and backs it up.
	r, err := WireZsh("/some/hook.zsh")
	if err != nil || !r.Changed {
		t.Fatalf("WireZsh #1: changed=%v err=%v", r.Changed, err)
	}
	data, _ := os.ReadFile(rc)
	if !strings.Contains(string(data), "/some/hook.zsh") {
		t.Errorf("zshrc not wired: %q", data)
	}
	if b, err := os.ReadFile(rc + ".hydrate.bak"); err != nil || string(b) != "# existing\n" {
		t.Errorf("backup missing/wrong: %q (err %v)", b, err)
	}

	// Second wire: already references hydrate -> no change.
	r2, err := WireZsh("/some/hook.zsh")
	if err != nil || r2.Changed {
		t.Errorf("WireZsh #2 should be a no-op, got changed=%v err=%v", r2.Changed, err)
	}
}

func TestWireTmuxIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	conf := filepath.Join(dir, "tmux", "tmux.conf")
	if err := os.MkdirAll(filepath.Dir(conf), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(conf, []byte("set -g mouse on\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := WireTmux()
	if err != nil || !r.Changed {
		t.Fatalf("WireTmux #1: changed=%v err=%v", r.Changed, err)
	}
	data, _ := os.ReadFile(conf)
	if !strings.Contains(string(data), "hydrate/segment") {
		t.Errorf("tmux.conf not wired: %q", data)
	}

	r2, err := WireTmux()
	if err != nil || r2.Changed {
		t.Errorf("WireTmux #2 should be a no-op, got changed=%v err=%v", r2.Changed, err)
	}
}
