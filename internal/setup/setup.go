// Package setup performs the system wiring for `hydrate init`: it materializes
// the embedded systemd units and zsh hook onto disk and enables the heartbeat.
// Everything is best-effort and reversible; nothing here edits the user's shell
// rc or tmux.conf.
package setup

import (
	"bytes"
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

// --- shell / tmux wiring (idempotent, with backups) ---

// WireResult reports what a wiring step did.
type WireResult struct {
	Path    string
	Changed bool // false means it was already wired (skipped)
}

// ZshrcPath resolves the interactive zsh rc: $ZDOTDIR/.zshrc, else ~/.zshrc.
func ZshrcPath() string {
	if z := os.Getenv("ZDOTDIR"); z != "" {
		return filepath.Join(z, ".zshrc")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".zshrc")
}

// TmuxConfPath resolves the tmux config: $XDG_CONFIG_HOME/tmux/tmux.conf if it
// exists, else ~/.tmux.conf.
func TmuxConfPath() string {
	p := filepath.Join(xdg("XDG_CONFIG_HOME", ".config"), "tmux", "tmux.conf")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tmux.conf")
}

// WireZsh appends a `source <hook>` line to the user's zshrc unless it already
// references hydrate. Backs the file up first.
func WireZsh(hookPath string) (WireResult, error) {
	path := ZshrcPath()
	data, _ := os.ReadFile(path)
	if bytes.Contains(data, []byte("hydrate")) {
		return WireResult{path, false}, nil
	}
	block := fmt.Sprintf("\n# hydrate — activity hook (added by `hydrate init`)\nsource %q\n", hookPath)
	if err := appendWithBackup(path, data, block); err != nil {
		return WireResult{path, false}, err
	}
	return WireResult{path, true}, nil
}

// WireTmux appends the status-bar segment block to tmux.conf unless it is
// already present. The appended block is itself runtime-idempotent (an if-shell
// guard) so re-sourcing the config won't duplicate the segment. Backs up first.
func WireTmux() (WireResult, error) {
	path := TmuxConfPath()
	data, _ := os.ReadFile(path)
	if bytes.Contains(data, []byte("hydrate/segment")) {
		return WireResult{path, false}, nil
	}
	block := "\n# hydrate — water segment (added by `hydrate init`)\n" +
		"set -g status-interval 5\n" +
		"set -g status-right-length 100\n" +
		"if-shell '! tmux show-options -g status-right | grep -q hydrate/segment' \\\n" +
		"  \"set -ag status-right ' #(cat \\${XDG_STATE_HOME:-\\$HOME/.local/state}/hydrate/segment 2>/dev/null)'\"\n"
	if err := appendWithBackup(path, data, block); err != nil {
		return WireResult{path, false}, err
	}
	return WireResult{path, true}, nil
}

// ReloadTmux best-effort re-sources a config into any running server.
func ReloadTmux(path string) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return
	}
	_ = exec.Command("tmux", "source-file", path).Run()
	_ = exec.Command("tmux", "refresh-client", "-S").Run()
}

func appendWithBackup(path string, existing []byte, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(existing) > 0 {
		if err := os.WriteFile(path+".hydrate.bak", existing, 0o644); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(block)
	return err
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
