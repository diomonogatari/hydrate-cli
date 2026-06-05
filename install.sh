#!/usr/bin/env bash
# hydrate installer — builds + installs the binary, the systemd --user heartbeat,
# and the zsh activity hook, then prints the two manual wiring steps it will not
# do for you (editing your shell rc and tmux.conf). Best-effort and idempotent.
set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local/bin}"
DO_BUILD=1
DO_TIMER=1

usage() {
  cat <<'EOF'
Usage: ./install.sh [options]

  --prefix DIR   Install the binary here (default: ~/.local/bin)
  --no-build     Skip `go build`; install the existing ./hydrate binary
  --no-timer     Skip installing/enabling the systemd --user heartbeat
  -h, --help     Show this help

This script never edits your shell rc or tmux.conf. It prints the exact lines
to add for the activity hook and the status-bar segment.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --prefix) PREFIX="$2"; shift 2 ;;
    --prefix=*) PREFIX="${1#*=}"; shift ;;
    --no-build) DO_BUILD=0; shift ;;
    --no-timer) DO_TIMER=0; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage; exit 2 ;;
  esac
done

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$here"

info() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarn:\033[0m %s\n' "$*"; }

# 1. Build -------------------------------------------------------------------
if [ "$DO_BUILD" -eq 1 ]; then
  command -v go >/dev/null 2>&1 || { echo "go is required (or pass --no-build)"; exit 1; }
  info "Building hydrate"
  go build -o hydrate .
fi
[ -x ./hydrate ] || { echo "no ./hydrate binary; build first or drop --no-build"; exit 1; }

# 2. Install binary ----------------------------------------------------------
info "Installing binary to $PREFIX/hydrate"
mkdir -p "$PREFIX"
install -m755 hydrate "$PREFIX/hydrate"
case ":$PATH:" in
  *":$PREFIX:"*) ;;
  *) warn "$PREFIX is not on your PATH" ;;
esac

# 3. zsh activity hook -------------------------------------------------------
hook_dir="${XDG_DATA_HOME:-$HOME/.local/share}/hydrate"
info "Installing zsh hook to $hook_dir/hydrate.zsh"
mkdir -p "$hook_dir"
install -m644 packaging/zsh/hydrate.zsh "$hook_dir/hydrate.zsh"

# 4. systemd --user heartbeat ------------------------------------------------
if [ "$DO_TIMER" -eq 1 ]; then
  if command -v systemctl >/dev/null 2>&1; then
    unit_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
    info "Installing systemd user units to $unit_dir"
    mkdir -p "$unit_dir"
    install -m644 packaging/systemd/hydrate.service "$unit_dir/hydrate.service"
    install -m644 packaging/systemd/hydrate.timer "$unit_dir/hydrate.timer"
    systemctl --user daemon-reload 2>/dev/null || warn "daemon-reload failed"
    if systemctl --user enable --now hydrate.timer 2>/dev/null; then
      info "Heartbeat enabled (hydrate.timer)"
    else
      warn "Could not enable hydrate.timer now; later run: systemctl --user enable --now hydrate.timer"
    fi
  else
    warn "systemctl not found; skipping the heartbeat timer"
  fi
fi

# 5. Manual wiring (never auto-edited) ---------------------------------------
cat <<EOF

hydrate installed. Two manual steps remain (left to you on purpose):

  1. tmux — add to your tmux.conf, AFTER any theme/tpm line:
       set -g status-interval 5
       set -ag status-right ' #(cat \${XDG_STATE_HOME:-\$HOME/.local/state}/hydrate/segment 2>/dev/null)'

  2. zsh — source the activity hook (so notifications respect "terminal in use"):
       source "$hook_dir/hydrate.zsh"
     It also adds: w = log a glass, ww = status.

Try it now:  hydrate log && hydrate
EOF
