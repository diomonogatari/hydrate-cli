#!/usr/bin/env bash
# hydrate installer — builds the binary, installs it, then hands off to the
# interactive `hydrate init` for setup (profile + heartbeat + shell/tmux wiring).
set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local/bin}"
DO_BUILD=1
DO_INIT=1

usage() {
  cat <<'EOF'
Usage: ./install.sh [options]

  --prefix DIR   Install the binary here (default: ~/.local/bin)
  --no-build     Skip `go build`; install the existing ./hydrate binary
  --no-init      Just install the binary; skip the interactive `hydrate init`
  -h, --help     Show this help
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --prefix) PREFIX="$2"; shift 2 ;;
    --prefix=*) PREFIX="${1#*=}"; shift ;;
    --no-build) DO_BUILD=0; shift ;;
    --no-init) DO_INIT=0; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage; exit 2 ;;
  esac
done

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$here"

info() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33mwarn:\033[0m %s\n' "$*"; }

if [ "$DO_BUILD" -eq 1 ]; then
  command -v go >/dev/null 2>&1 || { echo "go is required (or pass --no-build)"; exit 1; }
  info "Building hydrate"
  go build -o hydrate .
fi
[ -x ./hydrate ] || { echo "no ./hydrate binary; build first or drop --no-build"; exit 1; }

info "Installing binary to $PREFIX/hydrate"
mkdir -p "$PREFIX"
install -m755 hydrate "$PREFIX/hydrate"
case ":$PATH:" in
  *":$PREFIX:"*) ;;
  *) warn "$PREFIX is not on your PATH — add it to use \`hydrate\` directly" ;;
esac

if [ "$DO_INIT" -eq 1 ]; then
  echo
  exec "$PREFIX/hydrate" init
fi

echo "Done. Run \`hydrate init\` to configure."
