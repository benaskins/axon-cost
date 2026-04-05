# axon-cost

LLM inference cost tracking middleware for axon-talk.

## Module

- Module path: `github.com/benaskins/axon-cost`
- Project type: library (no main package)

## Build & Test

```bash
just test    # go test -race ./...
just vet     # go vet ./...
just build   # go build ./...
```

## Architecture

Single package (`cost/`) with four key types:

| Type | Purpose |
|------|---------|
| `Middleware` | Wraps `talk.LLMClient`, intercepts Chat calls, computes cost |
| `Aggregator` | Accumulates `CostRecord`s, computes totals by label/model |
| `RateTable` | Maps provider+model to per-million USD rates (embedded YAML) |
| `CostRecord` | Single-call cost result |

Read [AGENTS.md](./AGENTS.md) for full architecture, boundary map, and concurrency model.

## Constraints

- Do not modify `talk.LLMClient` — `Middleware` wraps it
- Only axon-talk and axon-fact as axon dependencies — no others
- `Middleware` and `Aggregator` must be safe for concurrent use
- Tests must not make real LLM calls — mock the underlying LLMClient
- No third-party assertion libraries — standard `testing` package only
