# Contributing

Thanks for taking a look! hydrate is a small, dependency-light Go CLI — easy to
build and quick to test.

## Prerequisites

- **Go 1.24+**
- Optional, for trying it end-to-end: tmux, a `systemd --user` session, and a
  freedesktop notification daemon.

## Build, test, lint

```bash
make build      # version-stamped binary (./hydrate)
make test       # all unit tests
make lint       # go vet + gofmt gate
make install    # install to ~/.local/bin
```

Or the plain Go equivalents: `go build ./...`, `go test ./...`, `go vet ./...`,
`gofmt -l .`.

## Project layout

See [Architecture › Package layout](docs/ARCHITECTURE.md#package-layout) for the
full map. In short: the domain logic (`internal/hydration`, `internal/notify`,
`internal/render`, `internal/store`) is pure and side-effect-free; `internal/cli`
and `internal/setup` hold the I/O and the interactive wizard.

## Conventions

- **Keep the hot path cheap.** The tmux status bar must never invoke the binary —
  it only reads the pre-rendered cache file.
- **Derive, don't store.** State comes from the append-only log; avoid mutable
  counters.
- **Pure where it counts.** New decision logic (urgency, notification policy)
  should be a pure function with table tests, separate from its I/O.
- **Minimal dependencies.** Prefer the standard library. The current third-party
  set is `BurntSushi/toml`, `godbus/dbus`, and `charmbracelet/huh` + `lipgloss`
  (for `hydrate init`).
- **Formatting & vetting are CI gates.** Run `make lint` before pushing.
- **Commits** follow a light Conventional Commits style: `feat:`, `fix:`,
  `docs:`, `build:`, `refactor:`.

## Releases

Releases are automated with [GoReleaser](https://goreleaser.com), triggered by
pushing a `vX.Y.Z` tag:

```bash
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

The `release` workflow builds Linux amd64/arm64 archives, generates checksums and
a changelog, and publishes a GitHub Release. Verify the whole pipeline locally
first — no publishing, no tag required:

```bash
goreleaser check                       # validate .goreleaser.yaml
goreleaser release --snapshot --clean  # full build into ./dist
```

CI (`.github/workflows/ci.yml`) runs gofmt, vet, build, and tests on every push
and pull request.
