# axon-cost — Working Instructions

## What This Is

`axon-cost` is a Go library (`github.com/benaskins/axon-cost`) that wraps `talk.LLMClient` to compute and aggregate LLM inference costs. It is a library only — no `main` package, no HTTP server, no CLI entrypoint.

## Module

- Module path: `github.com/benaskins/axon-cost`
- Project type: library
- Go version: 1.26

## Build & Test

```bash
just test    # go test -race ./...
just vet     # go vet ./...
just build   # go build ./...
```

Always run `just test` (not `go test ./...`) — the justfile enables the race detector.

## Constraints

Follow these strictly:

- **No real LLM calls in tests.** Mock `talk.LLMClient` using a struct with a `Chat` func field.
- **No writes outside `t.TempDir()`.** File-based tests (e.g. YAML loading) must use `t.TempDir()`.
- **No axon imports beyond axon-talk and axon-fact.** No axon-loop, axon-tool, axon-base, etc.
- **Do not modify `talk.LLMClient`.** `cost.Middleware` wraps it; the interface is owned by axon-talk.
- **Rate tables need no external config at runtime.** Use `DefaultRateTable()` (Go map literal) or `//go:embed`.
- **`Middleware` and `Aggregator` must be safe for concurrent use.** Use `sync.Mutex` / `sync.RWMutex`.
- **No third-party assertion libraries.** Standard `testing` package only.

## Key Types

| Type                   | File                     | Purpose                                      |
|------------------------|--------------------------|----------------------------------------------|
| `cost.Middleware`      | `cost/middleware.go`     | Wraps `talk.LLMClient`, intercepts Chat      |
| `cost.Aggregator`      | `cost/aggregator.go`     | Accumulates `CostRecord`s, computes totals   |
| `cost.RateTable`       | `cost/ratetable.go`      | Maps provider+model to per-million USD rates |
| `cost.CostRecord`      | `cost/types.go`          | Single-call cost result                      |
| `cost.CostSummary`     | `cost/aggregator.go`     | Aggregated totals with ByLabel / ByModel     |

## Practice

1. Read the plan in `plans/`. Pick the next incomplete step.
2. Write a failing test first, then make it pass, then clean up.
3. Run `just test` before committing. Fix all failures.
4. Stage only files for this step. One commit per step.
5. Use conventional commit messages: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `infra:`, `config:`.

## Plan

See `plans/2026-04-03-initial-build.md`.
