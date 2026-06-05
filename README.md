# 💧 hydrate

**A quiet hydration nudge for people who live in the terminal.**

[![CI](https://github.com/diomonogatari/hydrate-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/diomonogatari/hydrate-cli/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24%2B-00ADD8.svg?logo=go&logoColor=white)](go.mod)

hydrate watches your water intake from your tmux status bar — calm when you're on
track, impossible to miss when you're behind — and reaches you with a single
desktop notification *only* when you've stepped away. No popups mid-flow, no
streaks to maintain, no app to open. Just a glass of water when you need one.

![hydrate: setup, then a tmux status segment that escalates from calm blue through amber to a bright "DRINK WATER" — calmed the moment you log a glass](demo/hydrate.gif)

The segment escalates with urgency, and a glass calms it instantly:

```text
ok          💧 5/8                  calm blue · easy to ignore
due         💧 5/8                  a gentle amber nudge
overdue     💧 5/8 ·due·            bolder · harder to miss
critical    💧 DRINK WATER 5/8      bright · bold · blinking · pulsing
```

## Quick start

Install the binary, then run the guided setup:

```bash
go install github.com/diomonogatari/hydrate-cli/cmd/hydrate@latest
hydrate init
```

`hydrate init` tailors your goal, wires up your shell and tmux, and starts the
background heartbeat — no repo to clone. Then:

```bash
hydrate log      # or just `w`  → log a glass; the bar relaxes instantly
hydrate          # or `ww`      → today at a glance
```

No Go toolchain? Grab a prebuilt binary from [Releases](../../releases) and run
`hydrate init`, or clone the repo and run `./install.sh` (build + install + setup
in one).

## Why you might like it

- **🫧 A glance, not a popup.** One tmux segment that escalates from calm blue to
  a blinking block as you fall behind — always there, never in your face.
- **🔕 It respects your focus.** Desktop notifications fire only when you're
  *away* from the keyboard. While you're typing, it stays silent.
- **💧 Calm on action.** Log a drink and the urgency resets immediately.
- **🌙 Smart about your day.** It knows your waking hours and rolls over on your
  schedule — silent while you sleep, no 3am guilt trips.
- **📄 Yours in plaintext.** The log is greppable JSONL; the config is
  hand-editable TOML. No database, no lock-in.
- **🪶 Featherweight.** A single static Go binary — no runtime, no daemon beyond a
  60-second user timer, and no `notify-send` dependency.

## Commands

| Command | What it does |
| --- | --- |
| `hydrate init` | Interactive setup (profile, heartbeat, shell/tmux wiring) |
| `hydrate` | Today's intake, time since last drink, next due |
| `hydrate log [amount]` | Log a drink — `500`, `500ml`, `16oz`, `1l` (default: one glass) |
| `hydrate undo` | Remove today's most recent drink |
| `hydrate stats [--days N]` | History rollup (default: last 7 days) |
| `hydrate config [--edit]` | Show or edit your settings |
| `hydrate --help` | Everything else |

After setup, `w` logs a glass and `ww` shows status. Add `--json` to most
commands for scripting.

## Configuration

The first run writes a commented `~/.config/hydrate/config.toml` — daily goal,
glass size, waking window, units, and how aggressively to notify. Re-run
`hydrate init` for the guided version, or `hydrate config --edit` to tweak by
hand. See [Architecture](docs/ARCHITECTURE.md#configuration) for every knob.

## Requirements

- **Linux**, with tmux for the status segment and — for notifications — any
  freedesktop-compatible desktop (COSMIC, GNOME, KDE, dunst, mako, …).
- **Go 1.24+** for `go install` / building from source — or just grab a prebuilt
  binary from [Releases](../../releases).
- A `systemd --user` session for the background heartbeat (optional — the bar
  still updates whenever you log).

## Learn more

- 📐 **[Architecture & design](docs/ARCHITECTURE.md)** — how the heartbeat,
  segment cache, idle gate, and urgency model fit together.
- 🤝 **[Contributing](CONTRIBUTING.md)** — build, test, and cut a release.

## License

[MIT](LICENSE) © diomonogatari
