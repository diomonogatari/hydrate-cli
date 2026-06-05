# Architecture & design

How hydrate works under the hood, and the principles that shape it.

## Design principles

- **Tied to you, not a terminal.** State lives under XDG and is derived from an
  append-only log, so logging from any pane updates one shared source of truth
  and survives closing every terminal.
- **Don't break focus.** While you're actively typing, the only signal is a quiet
  tmux segment. Desktop notifications are reserved for when you're away.
- **Calm instantly on action.** Logging a drink resets urgency immediately.
- **Plaintext is the artifact.** The log is greppable JSONL; the config is a
  commented TOML file you can edit by hand.
- **No nagging.** No streaks, no guilt mechanics; it stays silent while you sleep.
- **Cheap hot path.** The status bar never invokes the binary — it reads a
  pre-rendered cache file.

## The moving parts

```text
  zsh hook ──► last_activity ─┐
                              │   (idle gate: only notify when away)
  systemd --user timer ──► hydrate tick ──► segment cache ──► tmux status bar
                              │
                              └─► org.freedesktop.Notifications (D-Bus)

  hydrate log / undo ──► append/rewrite log.jsonl ──► re-render segment ──► tmux refresh
```

- An append-only **JSONL log** is the single source of truth; every value is
  derived on demand, never stored as a mutable counter.
- A **`systemd --user` timer** runs `hydrate tick` roughly every 60s: it
  recomputes urgency, rewrites the **segment cache**, and — only when you've been
  away from the terminal — sends a desktop notification.
- The tmux status bar simply `cat`s the cache file (`status-right`), so rendering
  the segment costs nothing and never blocks the prompt.
- Notifications go straight over the **freedesktop D-Bus interface**
  (`org.freedesktop.Notifications`) — no `notify-send` binary required — and reuse
  `replaces_id` so a nudge updates in place rather than stacking.

## Deriving state from the log

`hydrate.Compute(cfg, events, now)` turns the raw event list into the full
picture. Nothing below is persisted:

- `today_ml` — sum of `ml` for events since the most recent `day_reset_hour`
  boundary (so a 1am glass counts toward the previous logical day).
- `interval` — the ideal gap between glasses across the waking window:
  `interval = (day_end_hour - day_start_hour) * 3600 / ceil(daily_goal_ml / glass_ml)`.
- `last_event` — `max(last drink today, today's day_start_hour)`. Starting the
  clock at `day_start` gives a one-interval grace period each morning.
- `since_last` — `now - last_event`.

## Urgency levels

`since_last` is mapped against `interval`. Outside the waking window the level is
pinned to `ok` — hydrate never nags while you're asleep.

| Level | Condition | Segment intent |
| --- | --- | --- |
| `ok` | `since_last < interval`, or outside waking window | subtle blue |
| `due` | `interval ≤ since_last < 1.5×` | gentle amber |
| `overdue` | `1.5× ≤ since_last < 2.5×` | bold, harder to miss |
| `critical` | `since_last ≥ 2.5×` | bright bg · bold · blink · pulse |

At `critical` the segment alternates between two forms each heartbeat (different
background shade and leading glyph) to add motion — peripheral vision keys on
motion and luminance contrast far more than colour.

## The notification gate

`hydrate tick` decides whether to notify with a pure function
(`notify.Decide`) so the policy is fully unit-tested:

1. Never below the configured `notify_min_level`.
2. Never while the terminal is **in use** — `now - last_activity < idle_threshold_sec`.
   The `last_activity` timestamp is stamped by the zsh hook on every prompt and
   command. (No hook installed → treated as "away", so notifications still work,
   just without the typing-suppression guarantee.)
3. Never while the terminal is the **focused window**, when that can be
   determined. The focus probe is best-effort and returns "unknown" on Wayland or
   when tools are missing — it never blocks a notification on its own.
4. Otherwise fire on **escalation** (level rose past the last-notified level), or
   once the **cooldown** has elapsed. Logging water re-arms escalation.

## tmux integration

The segment is a self-contained styled string written to the cache by `tick`,
`log`, and `undo`. The status bar reads it with:

```tmux
set -g status-interval 5
set -g status-right-length 100   # the default (40) truncates the segment
set -ag status-right ' #(cat ${XDG_STATE_HOME:-$HOME/.local/state}/hydrate/segment 2>/dev/null)'
```

`hydrate init` appends this idempotently (guarded by an `if-shell` so re-sourcing
your config never duplicates it) and after any theme/tpm line so a theme can't
overwrite `status-right`.

## Configuration

`${XDG_CONFIG_HOME:-~/.config}/hydrate/config.toml`, written commented on first
run:

| Key | Default | Meaning |
| --- | --- | --- |
| `daily_goal_ml` | `2000` | Target volume per day |
| `glass_ml` | `250` | Default amount for `hydrate log` |
| `day_start_hour` / `day_end_hour` | `7` / `23` | Waking window (local); outside it, urgency stays calm |
| `day_reset_hour` | `4` | When the logical day rolls over |
| `idle_threshold_sec` | `600` | Shell idle before you count as "away" |
| `notify_min_level` | `"overdue"` | Lowest urgency that may notify (`due`/`overdue`/`critical`) |
| `notify_cooldown_sec` | `1800` | Minimum gap between notifications (escalation overrides it) |
| `units` | `"ml"` | `ml` or `oz`, display only |

Unknown keys are ignored, so configs survive across versions.

## On-disk layout

| Path | Purpose |
| --- | --- |
| `~/.config/hydrate/config.toml` | User settings |
| `~/.local/state/hydrate/log.jsonl` | Append-only drink log (`{"ts":…,"ml":…}`) |
| `~/.local/state/hydrate/segment` | Pre-rendered tmux string (cache) |
| `~/.local/state/hydrate/last_activity` | Shell-activity timestamp (idle gate) |
| `~/.local/state/hydrate/notify_state.json` | Last notification (cooldown/escalation) |

Nothing secret is ever stored — only timestamps and millilitres.

## Package layout

| Package | Responsibility |
| --- | --- |
| `internal/paths` | XDG path resolution |
| `internal/config` | Load/save the commented TOML config |
| `internal/store` | Append-only JSONL log, segment cache, atomic writes |
| `internal/hydration` | Derived state, urgency levels, daily rollups (pure) |
| `internal/render` | tmux segment styling |
| `internal/notify` | Notification policy (pure) + D-Bus delivery |
| `internal/focus` | Best-effort window-focus probe |
| `internal/setup` | `hydrate init` system wiring (systemd, hook, shell/tmux) |
| `internal/cli` | Command dispatch and the interactive wizard |

The domain logic (`hydration`, `notify` policy, `render`, `store`) is pure and
side-effect-free, which is why it carries the bulk of the test suite.
