# Linting policy

This repository uses [golangci-lint](https://golangci-lint.run/) **v2** with a configuration tuned for **green CI** and meaningful signal for contributors.

## Running locally

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
golangci-lint run --timeout=5m
```

Use **Go 1.26+** (matches `go.mod`).

## What is enforced today

The active linters focus on correctness and safety (for example `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`, `gosec`, formatters). See `.golangci.yml` for the exact list.

## Known debt (not enforced yet)

Broader stylistic rules — **`goconst`**, **`godot`**, **`dupl`**, **`revive` package-comments**, **`err113`**, strict **`mnd`**, and heavy **`gocritic`** — are **not** enabled yet. Turning them on currently produces hundreds of findings across legacy helpers (including `internal/logging`). They are candidates for incremental cleanup; contributions that chip away at that debt are welcome.

If you run a stricter profile locally, please do not force-enable those linters in CI without a coordinated cleanup PR.
