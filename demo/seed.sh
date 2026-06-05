#!/usr/bin/env bash
# Sandboxed, side-effect-free seed for the demo GIF — sourced by demo/hydrate.tape.
# Temp dirs + a sandboxed TMUX_TMPDIR + a stubbed systemctl mean the recording
# (the init wizard AND the tmux segment) never touches your real config, units,
# shell, or tmux server.

export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"
export XDG_STATE_HOME="$(mktemp -d)"
export XDG_CONFIG_HOME="$(mktemp -d)"
export XDG_DATA_HOME="$(mktemp -d)"
export ZDOTDIR="$XDG_CONFIG_HOME/zsh"
export TMUX_TMPDIR="$XDG_STATE_HOME/tmux"   # isolate the demo's tmux server
export PS1=$'\033[38;5;39m❯\033[0m '        # a calm blue prompt

mkdir -p "$XDG_STATE_HOME/hydrate" "$XDG_CONFIG_HOME/hydrate" \
         "$ZDOTDIR" "$XDG_CONFIG_HOME/tmux" "$TMUX_TMPDIR"
: > "$ZDOTDIR/.zshrc"
: > "$XDG_CONFIG_HOME/tmux/tmux.conf"

# Stub systemctl so `hydrate init`'s heartbeat step is a no-op while recording.
STUB="$XDG_STATE_HOME/.bin"; mkdir -p "$STUB"
printf '#!/bin/sh\nexit 0\n' > "$STUB/systemctl"; chmod +x "$STUB/systemctl"
export PATH="$STUB:$PATH"

# Wide waking window keeps the time-lapse aging robust regardless of record time.
cat > "$XDG_CONFIG_HOME/hydrate/config.toml" <<'CFG'
daily_goal_ml       = 2000
glass_ml            = 250
day_start_hour      = 0
day_end_hour        = 23
day_reset_hour      = 0
idle_threshold_sec  = 600
notify_min_level    = "critical"
notify_cooldown_sec = 99999
units               = "ml"
CFG

# 3 glasses today (recent → calm) + six varied past days for the stats view.
now=$(date +%s); D=86400
{
  for m in 200 130 70; do printf '{"ts":%d,"ml":250}\n' "$((now - m*60))"; done
  printf '{"ts":%d,"ml":2000}\n' "$((now-1*D))"
  printf '{"ts":%d,"ml":1750}\n' "$((now-2*D))"
  printf '{"ts":%d,"ml":2000}\n' "$((now-3*D))"
  printf '{"ts":%d,"ml":1250}\n' "$((now-4*D))"
  printf '{"ts":%d,"ml":1500}\n' "$((now-5*D))"
  printf '{"ts":%d,"ml":2000}\n' "$((now-6*D))"
} > "$XDG_STATE_HOME/hydrate/log.jsonl"
printf '%s' "$now" > "$XDG_STATE_HOME/hydrate/last_activity"
hydrate tick >/dev/null 2>&1   # write the initial segment cache (calm 3/8)
