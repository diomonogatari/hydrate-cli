# hydrate

A quiet, terminal-independent hydration nudge for people who live in a terminal
but rarely *look* at it.

`hydrate` tracks your water intake in a plaintext log and surfaces a single,
unobtrusive signal in your tmux status bar. When you step away from the keyboard
and fall behind, it reaches you with one calm desktop notification — and the
moment you log a drink, the urgency resets.

## Design principles

- **Tied to you, not a terminal.** State lives under XDG, derived from an
  append-only log — logging from any pane updates one shared source of truth and
  survives closing every terminal.
- **Don't break focus.** While you're actively typing, the only signal is a
  quiet tmux segment. Desktop notifications are reserved for when you're away.
- **Calm instantly on action.** Logging a drink resets urgency immediately.
- **Plaintext is the artifact.** The log is greppable JSONL; the config is a
  commented TOML file you can edit by hand.
- **No nagging.** No streaks, no guilt mechanics; it stays silent while you sleep.

## How it works

```text
  shell hook ──► last_activity ─┐
                                │   (idle gate: only notify when away)
  systemd --user timer ──► hydrate tick ──► segment cache ──► tmux status bar
                                │
                                └─► org.freedesktop.Notifications (D-Bus)
```

- An append-only **JSONL log** is the single source of truth; all state is
  derived, never stored as a counter.
- A **`systemd --user` timer** runs `hydrate tick` ~every 60s: it recomputes
  urgency, rewrites a pre-rendered **tmux segment cache**, and — only when you've
  been away from the terminal — sends a desktop notification.
- The status bar just `cat`s the cache file, so the **hot path never invokes the
  binary**.
- Notifications go straight over the **freedesktop D-Bus interface** (no
  `notify-send` dependency) and work natively on COSMIC, GNOME, KDE, dunst, mako…

## Install

Requires Go 1.24+.

```bash
git clone https://github.com/diomonogatari/hydrate-cli
cd hydrate-cli
./install.sh         # builds, installs the binary + zsh hook + systemd timer
```

`install.sh` is best-effort and idempotent. It never edits your shell rc or
tmux.conf — it prints the two lines to add yourself. Flags: `--prefix DIR`,
`--no-timer`, `--no-build`.

Just the binary, no system wiring:

```bash
go install github.com/diomonogatari/hydrate-cli@latest   # to $GOBIN
# or: make install      (installs to ~/.local/bin)
```

## Usage

```bash
hydrate              # today's intake, time since last drink, next due
hydrate log          # log one glass (the configured default size)
hydrate log 500      # log 500 ml
hydrate log 16oz     # also accepts oz and l (e.g. 1l, 0.5l)
hydrate undo         # remove the most recent drink logged today
hydrate stats        # last 7 days: per-day bar, %, average, goal-met count
hydrate stats --days 14
hydrate config       # show resolved settings and their file path
hydrate config --edit  # open the config in $EDITOR
hydrate tick         # the heartbeat (normally run by the timer; safe by hand)
hydrate segment      # print the styled tmux string (debugging)
```

Add `--json` to `status`, `log`, `undo`, `tick`, or `stats` for machine-readable
output. With the shell hook installed, `w` = `hydrate log` and `ww` = `hydrate
status`.

## tmux status bar

The status bar reads a **pre-rendered cache file** — it never invokes the binary,
so there's no hot-path cost. Add this to your `tmux.conf`, **after** any
theme/tpm line (so a theme can't overwrite `status-right`):

```tmux
set -g status-interval 5
set -ag status-right ' #(cat ${XDG_STATE_HOME:-$HOME/.local/state}/hydrate/segment 2>/dev/null)'
```

The segment escalates with urgency — subtle blue when `ok`, through amber when
`overdue`, to a bright, bold, blinking, pulsing block at `critical` (built to
catch peripheral vision). Set `nuclear_escalation = true` to additionally
recolor the whole bar at `critical`.

## Shell hook (recommended)

Sourcing the hook lets hydrate tell whether the terminal is *in use*, so it never
notifies while you're typing. It is output-silent (Powerlevel10k instant-prompt
safe) and fork-free:

```zsh
source "${XDG_DATA_HOME:-$HOME/.local/share}/hydrate/hydrate.zsh"
```

## Configuration

On first run, `hydrate` writes a commented config to
`${XDG_CONFIG_HOME:-~/.config}/hydrate/config.toml`:

| Key | Default | Meaning |
| --- | --- | --- |
| `daily_goal_ml` | `2000` | Target volume per day |
| `glass_ml` | `250` | Default amount for `hydrate log` |
| `day_start_hour` / `day_end_hour` | `7` / `23` | Waking window (local); outside it, urgency stays calm |
| `day_reset_hour` | `4` | When the logical day rolls over (a 1am glass counts to the prior day) |
| `idle_threshold_sec` | `600` | Shell idle before you count as "away" |
| `notify_min_level` | `"overdue"` | Lowest urgency that may notify (`due`/`overdue`/`critical`) |
| `notify_cooldown_sec` | `1800` | Minimum gap between notifications (escalation overrides it) |
| `units` | `"ml"` | `ml` or `oz`, display only |
| `nuclear_escalation` | `false` | Recolor the whole tmux bar at `critical` (intrusive, opt-in) |

## Urgency levels

`since_last` is measured against an `interval` derived from your goal and waking
window (`interval = waking_seconds / glasses_needed`):

| Level | When |
| --- | --- |
| `ok` | `since_last < interval`, or outside the waking window |
| `due` | `interval ≤ since_last < 1.5×` |
| `overdue` | `1.5× ≤ since_last < 2.5×` |
| `critical` | `since_last ≥ 2.5×` |

## Data & files

| Path | Purpose |
| --- | --- |
| `~/.config/hydrate/config.toml` | User settings |
| `~/.local/state/hydrate/log.jsonl` | Append-only drink log (`{"ts":…,"ml":…}`) |
| `~/.local/state/hydrate/segment` | Pre-rendered tmux string (cache) |
| `~/.local/state/hydrate/last_activity` | Shell-activity timestamp (idle gate) |
| `~/.local/state/hydrate/notify_state.json` | Last notification (cooldown/escalation) |

Nothing secret is ever stored — only timestamps and millilitres.

## Development

```bash
make build      # version-stamped binary
make test       # all unit tests
make lint       # go vet + gofmt gate
```

Releases are cut by pushing a `vX.Y.Z` tag: GitHub Actions runs GoReleaser and
attaches Linux amd64/arm64 archives.

## License

[MIT](LICENSE).
