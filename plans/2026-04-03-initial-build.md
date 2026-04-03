# axon-cost — Initial Build Plan
# 2026-04-03

Each step is commit-sized. Execute via `/iterate`.

## Step 1 — Scaffold library module and project files

Initialise the Go module as `github.com/(...)/axon-cost` (library, no main package). Create `go.mod` with dependencies on axon-talk and axon-fact. Add `justfile` with `build`, `test`, and `lint` targets. Add `README.md`, `AGENTS.md`, and `CLAUDE.md` stubs. Verify: `go build ./...` passes on an empty package placeholder.

Commit: `infra: scaffold axon-cost library module with go.mod and justfile`

## Step 2 — Define CostRecord and core types

Create `cost/types.go` with: `CostRecord` struct (Model, Provider, InputTokens, OutputTokens, InputCost, OutputCost, TotalCost float64, Label string, Timestamp time.Time); `RateEntry` struct (InputPerMillion, OutputPerMillion float64); `RateKey` struct (Provider, Model string). No logic yet, just type definitions. Verify: `go vet ./...` passes.

Commit: `feat: define CostRecord, RateEntry, and RateKey types`

## Step 3 — Implement RateTable with built-in rates and YAML loader

Create `cost/ratetable.go`. Implement `RateTable` (map[RateKey]RateEntry protected by sync.RWMutex). Add `DefaultRateTable()` that returns a table pre-populated from a Go map literal with built-in entries: OpenRouter/Qwen3.5-122B ($0.26 in / $2.08 out), Anthropic/claude-sonnet-4-6 ($3/$15), Anthropic/claude-haiku-4-5 ($0.80/$4). Add `(rt *RateTable) Lookup(provider, model string) (RateEntry, bool)`. Add `(rt *RateTable) LoadYAML(r io.Reader) error` to merge additional entries from YAML (format: list of {provider, model, input_per_million, output_per_million}). Add `embed.go` with `//go:embed rates.yaml` for an optional bundled YAML file. Write unit tests in `cost/ratetable_test.go`: test built-in lookup, YAML merge, unknown model returns false. Tests write YAML to `t.TempDir()` for file-based tests.

Commit: `feat: implement RateTable with built-in rates and YAML loader`

## Step 4 — Implement cost calculation and token estimation

Create `cost/calc.go`. Implement `Calculate(provider, model string, inputTokens, outputTokens int, rt *RateTable) (CostRecord, error)`. Returns error when rate is not found. Input/output cost = tokens / 1_000_000 * rate. TotalCost = InputCost + OutputCost. Timestamp set to time.Now(). Also implement `EstimateTokens(s string, tokensPerChar float64) int` for fallback estimation (default ratio 0.25 chars→tokens). Unit tests: deterministic arithmetic, known inputs produce expected USD costs, estimation rounds correctly.

Commit: `feat: implement cost calculation from token counts and rate table`

## Step 5 — Implement Middleware wrapping talk.LLMClient

Create `cost/middleware.go`. Define `Middleware` struct that holds: inner `talk.LLMClient`, `*RateTable`, label string, optional `*Aggregator`, optional `fact.EventStore`, tokensPerChar float64. Implement `Chat(ctx, req) (resp, err)` that: calls the inner client, extracts token counts from the response (with EstimateTokens fallback if counts are zero), calls Calculate, attaches the label, records to Aggregator if set, emits to EventStore if set (event type "inference.cost", payload JSON of CostRecord). Implement constructor `New(inner talk.LLMClient, rt *RateTable, opts ...Option) *Middleware` and functional options: `WithLabel(string)`, `WithAggregator(*Aggregator)`, `WithEventStore(fact.EventStore)`, `WithTokensPerChar(float64)`. Middleware is thread-safe via its inner state being immutable after construction (aggregator and event store handle their own locking). Unit tests use a mock LLMClient (struct with a Chat func field). Test: label attached, aggregator receives record, EventStore.Append called with correct event type. No real LLM calls.

Commit: `feat: implement Middleware wrapping talk.LLMClient`

## Step 6 — Implement Aggregator and CostSummary with markdown rendering

Create `cost/aggregator.go`. Implement `Aggregator` struct with a `sync.Mutex`, slice of `CostRecord`, and running totals. Implement `(a *Aggregator) Record(r CostRecord)` to append and update totals under lock. Implement `(a *Aggregator) Summary() CostSummary`. `CostSummary` holds: TotalInputTokens int, TotalOutputTokens int, TotalCost float64, ByLabel map[string]float64, ByModel map[string]float64. Implement `(cs CostSummary) MarkdownTable() string` rendering a markdown table with columns: Label/Model, Input Tokens, Output Tokens, Cost (USD). Unit tests: concurrent Record calls (use t.Parallel and goroutines), Summary returns correct totals, MarkdownTable produces valid markdown header + rows.

Commit: `feat: implement Aggregator with per-label and per-model breakdown`

## Step 7 — Implement budget alert callback

Extend `Aggregator` to support a budget threshold. Add `WithBudget(threshold float64, cb func(current, threshold float64)) AggregatorOption` constructor option. After each `Record` call, if running TotalCost exceeds threshold and the callback is non-nil, fire the callback in a goroutine (non-blocking). Add a `budgetFired bool` flag under the same mutex so the callback only fires once per threshold crossing (re-arms never — caller resets by creating a new aggregator). Unit tests: callback fires when threshold exceeded, does not fire when under threshold, does not fire twice.

Commit: `feat: implement budget alert callback on Aggregator`

## Step 8 — Harden axon-fact event emission in Middleware

Harden the EventStore emission path in Middleware. Ensure `fact.Event` is constructed with: Type="inference.cost", Payload = JSON-marshalled CostRecord, ID = new UUID (use google/uuid or crypto/rand), Timestamp from CostRecord. If EventStore.Append returns an error, log to stderr (do not propagate — cost tracking must not break inference). Add an integration-style test using a fake in-memory EventStore (implement `AppendFunc` adapter in test file). Verify event type, payload round-trips, and that an Append error does not cause Chat to return an error.

Commit: `feat: emit inference.cost events to axon-fact EventStore`

## Step 9 — Add race-detector concurrency tests

Create `cost/concurrent_test.go`. Spin up 50 goroutines each calling `middleware.Chat` with the mock LLMClient. Assert Aggregator.Summary().TotalCost equals 50 × single-call cost (deterministic mock always returns fixed token counts). Use `go test -race ./...` as the verification command in the justfile `test` target. Confirm zero race detector findings.

Commit: `test: add concurrent safety tests for Middleware and Aggregator`

## Step 10 — Write README, AGENTS.md, and CLAUDE.md

Write `README.md` covering: what axon-cost is, quick-start code snippet (New + WithLabel + WithAggregator + Summary), built-in rate table listing, YAML rate table format. Write `AGENTS.md` with module selections (axon-talk, axon-fact), boundary map, and dependency graph. Write `CLAUDE.md` with working instructions: run `just test` (includes -race), constraints summary (no real LLM calls, no writes outside t.TempDir, no axon imports beyond talk+fact). Verify: `just build` and `just test` both pass cleanly.

Commit: `docs: write README, AGENTS.md, and CLAUDE.md`

