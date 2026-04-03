# axon-cost

Initialise the Go module as `github.com/(...)/axon-cost` (library, no main package). Create `go.mod` with dependencies on axon-talk and axon-fact. Add `justfile` with `build`, `test`, and `lint` targets. Add `README.md`, `AGENTS.md`, and `CLAUDE.md` stubs. Verify: `go build ./...` passes on an empty package placeholder.

## Build & Test

```bash
go test ./...
go vet ./...
just build     # builds to bin/axon-cost
just install   # copies to ~/.local/bin/axon-cost
```

## Module Selections

- **axon-talk**: axon-cost wraps the talk.LLMClient interface to intercept Chat calls and capture token counts. axon-talk is the core dependency being wrapped. (deterministic)
- **axon-fact**: Cost events are emitted as axon-fact compatible events (Event type, EventStore). CostRecord payloads are wrapped in fact.Event with type "inference.cost". (deterministic)

## Deterministic / Non-deterministic Boundary

| From | To | Type |
|------|----|------|
| cost.Middleware | axon-talk.LLMClient | non-det |
| cost.Middleware | cost.RateTable | det |
| cost.Middleware | cost.Aggregator | det |
| cost.Middleware | axon-fact.EventStore | det |
| cost.Aggregator | cost.CostSummary | det |
| cost.RateTable | embedded YAML / Go map | det |

