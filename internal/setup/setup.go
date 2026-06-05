// Package setup performs the system wiring for `hydrate init`: it materializes
// the embedded systemd units and zsh hook onto disk and enables the heartbeat.
// Everything is best-effort and reversible; nothing here edits the user's shell
// rc or tmux.conf.
package setup

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// Embedded asset paths (relative to the embedded FS root).
const (
	serviceAsset = "packaging/systemd/hydrate.service"
	timerAsset   = "packaging/systemd/hydrate.timer"
	hookAsset    = "packaging/zsh/hydrate.zsh"
)

// SystemdUserDir is where the user's systemd units live.
func SystemdUserDir() string {
	return filepath.Join(xdg("XDG_CONFIG_HOME", ".config"), "systemd", "user")
}

// HookPath is where the zsh activity hook is installed.
func HookPath() string {
	return filepath.Join(xdg("XDG_DATA_HOME", ".local", "share"), "hydrate", "hydrate.zsh")
}

// InstallUnits writes the systemd service+timer from the embedded assets into
// the user unit directory and returns that directory.
func InstallUnits(assets fs.FS) (string, error) {
	dir := SystemdUserDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dir, err
	}
	for _, a := range []struct{ asset, name string }{
		{serviceAsset, "hydrate.service"},
		{timerAsset, "hydrate.timer"},
	} {
		data, err := fs.ReadFile(assets, a.asset)
		if err != nil {
			return dir, fmt.Errorf("reading embedded %s: %w", a.asset, err)
		}
		if err := os.WriteFile(filepath.Join(dir, a.name), data, 0o644); err != nil {
			return dir, err
		}
	}
	return dir, nil
}

// EnableTimer reloads the user manager and enables+starts the heartbeat timer.
func EnableTimer() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl not found")
	}
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload: %v: %s", err, out)
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", "hydrate.timer").CombinedOutput(); err != nil {
		return fmt.Errorf("enable --now hydrate.timer: %v: %s", err, out)
	}
	return nil
}

// InstallHook writes the zsh activity hook from the embedded assets and returns
// its path.
func InstallHook(assets fs.FS) (string, error) {
	path := HookPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, err
	}
	data, err := fs.ReadFile(assets, hookAsset)
	if err != nil {
		return path, fmt.Errorf("reading embedded %s: %w", hookAsset, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return path, err
	}
	return path, nil
}

func xdg(envVar string, fallback ...string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(append([]string{home}, fallback...)...)
}
