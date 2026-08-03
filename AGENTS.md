# AGENTS.md — Tether

Go CLI for agent skill management. Module: `github.com/zainhaider-123/tether-store`, Go 1.26.4.

## Build & test

```bash
make build          # bin/tether (default)
make test
make run ARGS="ls"  # or: make dev ARGS="search frontend"
```

Equivalent without Make: `go build -o bin/tether ./cmd/tether` and `go test ./...`.

CI: `.github/workflows/ci.yml` (vet/test/build). Releases: `.github/workflows/release.yml` + `.goreleaser.yaml` (tag `v*`). `mise.toml` only pins `go = "latest"`. See `make help` for all targets (`vet`, `tidy`, `install`, `clean`).

## Architecture

- **Entrypoint:** `cmd/tether/main.go` calls `internal/cli/cli.go` (Cobra command tree).
- **Internal packages:** `config`, `store`, `agents`, `link`, `prompt`, `search`, `source`, `paths`.
- `internal/agents/agents.go` is the **agent registry** — add new agents there (maps ID → project/global skill paths).
- `internal/store/store.go` clones git repos and expects a `SKILL.md` to identify a skill directory.
- `internal/paths/paths.go` resolves XDG config/data dirs for tether.

## Testing conventions

Tests mock the real filesystem and XDG paths to avoid touching the user's home dir:

```go
t.Setenv("HOME", home)
t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
```

Use `t.TempDir()` for all test roots. Both `config` and `source` have `_test.go` files; follow the same `_test` package style (`package config_test`).
