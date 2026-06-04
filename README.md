# hydrate

A quiet, terminal-independent hydration nudge for people who live in a terminal
but rarely *look* at it.

`hydrate` tracks your water intake in a plaintext log and surfaces a single,
unobtrusive signal in your tmux status bar. When you step away from the keyboard
and fall behind, it can (later phases) reach you with one calm desktop
notification — and the moment you log a drink, the urgency resets.

> **Status: early.** Phase 1 (the core CLI) is implemented and usable today.
> Background heartbeat, desktop notifications, and shell integration land in
> later phases — see the [roadmap](#roadmap).

## Design principles

- **Tied to you, not a terminal.** State lives under XDG, derived from an
  append-only log — logging from any pane updates one shared source of truth.
- **Don't break focus.** While you're actively typing, the only signal is a
  quiet tmux segment. Notifications are reserved for when you're away.
- **Calm instantly on action.** Logging a drink resets urgency immediately.
- **Plaintext is the artifact.** The log is greppable JSONL; the config is a
  commented TOML file you can edit by hand.
- **No nagging.** No streaks, no guilt mechanics; it stays silent while you sleep.

## Install

Requires Go 1.24+.

```bash
go install github.com/diomonogatari/hydrate-cli@latest   # installs `hydrate` to $GOBIN
# or build from a clone:
git clone https://github.com/diomonogatari/hydrate-cli
cd hydrate-cli
go build -o hydrate .
install -m755 hydrate ~/.local/bin/hydrate
```

## Usage

```bash
hydrate              # today's intake, time since last drink, next due
hydrate log          # log one glass (the configured default size)
hydrate log 500      # log 500 ml
hydrate log 16oz     # also accepts oz and l (e.g. 1l, 0.5l)
hydrate undo         # remove the most recent drink logged today
hydrate config       # show resolved settings and their file path
hydrate config --edit  # open the config in $EDITOR
hydrate segment      # print the styled tmux string (debugging)
```

Add `--json` to `status`, `log`, or `undo` for machine-readable output.

## tmux status bar (manual, Phase 1)

The status bar reads a **pre-rendered cache file** — it never invokes the binary,
so there's no hot-path cost. Add this to your `tmux.conf` (anywhere in
`status-right`):

```tmux
set -g status-interval 5
set -ag status-right ' #(cat ${XDG_STATE_HOME:-$HOME/.local/state}/hydrate/segment 2>/dev/null)'
```

The cache is (re)written whenever you `hydrate log` / `hydrate undo`. Until the
background heartbeat lands (Phase 2), the segment refreshes on those actions
rather than on its own.

## Configuration

On first run, `hydrate` writes a commented config to
`${XDG_CONFIG_HOME:-~/.config}/hydrate/config.toml`:

| Key | Default | Meaning |
| --- | --- | --- |
| `daily_goal_ml` | `2000` | Target volume per day |
| `glass_ml` | `250` | Default amount for `hydrate log` |
| `day_start_hour` / `day_end_hour` | `7` / `23` | Waking window (local) |
| `day_reset_hour` | `4` | When the logical day rolls over |
| `idle_threshold_sec` | `600` | Shell idle before "away" (later phases) |
| `notify_min_level` | `"overdue"` | Lowest urgency that may notify (later phases) |
| `notify_cooldown_sec` | `1800` | Minimum gap between notifications |
| `units` | `"ml"` | `ml` or `oz`, display only |

## Data & files

| Path | Purpose |
| --- | --- |
| `~/.config/hydrate/config.toml` | User settings |
| `~/.local/state/hydrate/log.jsonl` | Append-only drink log (`{"ts":…,"ml":…}`) |
| `~/.local/state/hydrate/segment` | Pre-rendered tmux string (cache) |

Nothing secret is ever stored — only timestamps and millilitres.

## Roadmap

- **Phase 1 — Core CLI** ✅ log / status / undo / config, JSONL log, urgency
  levels, tmux segment rendering.
- **Phase 2 — Heartbeat.** A `systemd --user` timer recomputes state and refreshes
  the segment on its own, so it escalates without you logging.
- **Phase 3 — Notifications.** Desktop notifications via the freedesktop D-Bus
  interface (works natively on COSMIC/GNOME/KDE), gated by a shell-activity check
  so they only fire when you're away from the terminal.
- **Phase 4 — Peripheral-vision escalation.** Bright/bold/blink + motion at the
  `critical` level.
- **Phase 5 — Polish.** History rollups (`hydrate stats`), shell aliases, optional
  prompt segment, packaged release binaries.

## License

[MIT](LICENSE).
