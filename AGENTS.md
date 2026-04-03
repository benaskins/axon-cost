# axon-cost — Architecture

## What This Is

`axon-cost` is a Go library that wraps `talk.LLMClient` to intercept `Chat` calls, compute USD inference cost, aggregate totals, and emit `inference.cost` events to an `axon-fact` EventStore. It has no HTTP server, no `main` package, and no CLI entrypoint.

## Module Selections

| Module     | Role                                                                                          | Boundary type |
|------------|-----------------------------------------------------------------------------------------------|---------------|
| axon-talk  | Provides `LLMClient` interface. `cost.Middleware` wraps it — does not modify the interface.   | non-det       |
| axon-fact  | Provides `Event` and `EventStore`. Cost records are emitted as `inference.cost` fact events.  | det           |

No other axon modules are used.

## Boundary Map

| From              | To                     | Type    | Notes                                          |
|-------------------|------------------------|---------|------------------------------------------------|
| caller            | cost.Middleware.Chat   | non-det | Middleware implements talk.LLMClient           |
| cost.Middleware   | inner talk.LLMClient   | non-det | Delegates to the wrapped client                |
| cost.Middleware   | cost.RateTable         | det     | Lookup is a pure map read                      |
| cost.Middleware   | cost.Aggregator        | det     | Record is a mutex-guarded append               |
| cost.Middleware   | axon-fact.EventStore   | det     | Append is called with a JSON-marshalled record |
| cost.Aggregator   | cost.CostSummary       | det     | Summary is a pure aggregation                  |
| cost.RateTable    | embedded YAML / Go map | det     | Rates loaded at init or via LoadYAML           |

## Dependency Graph

```
caller
  └── cost.Middleware  (implements talk.LLMClient)
        ├── talk.LLMClient  (inner, non-det)
        ├── cost.RateTable  (rates lookup)
        ├── cost.Aggregator (optional, running totals)
        └── fact.EventStore (optional, event emission)
              └── axon-fact
```

## Concurrency

Both `Middleware` and `Aggregator` are safe for concurrent use:

- `Middleware` fields are immutable after construction. Shared state (`Aggregator`, `EventStore`) manage their own locking.
- `Aggregator` uses a `sync.Mutex` around all record writes and reads.
- `RateTable` uses a `sync.RWMutex` (read-heavy; `LoadYAML` acquires a write lock).

## Event Schema

Each `Chat` call emits a `fact.Event` with:

- `Type`: `"inference.cost"`
- `Data`: JSON-marshalled `cost.CostRecord`
- `ID`: random UUID v4 (crypto/rand)
- `OccurredAt`: `CostRecord.Timestamp`

Emission errors are logged to stderr and do not propagate — cost tracking must not break inference.
