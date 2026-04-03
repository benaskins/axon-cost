# axon-cost

`axon-cost` is an axon library that wraps `talk.LLMClient` to track inference costs per call. It computes USD cost from token counts against a built-in (or custom) rate table, aggregates totals across calls, and optionally emits `inference.cost` events to an `axon-fact` EventStore.

## Quick Start

```go
import (
    "context"
    "fmt"

    "github.com/benaskins/axon-cost/cost"
    talk "github.com/benaskins/axon-talk"
)

func main() {
    rt := cost.DefaultRateTable()
    agg := cost.NewAggregator()

    // Wrap your existing LLMClient
    client := cost.New(
        yourTalkClient,
        rt,
        cost.WithProvider("Anthropic"),
        cost.WithLabel("my-agent"),
        cost.WithAggregator(agg),
    )

    // Use client as a normal talk.LLMClient
    err := client.Chat(ctx, &talk.Request{
        Model:    "claude-sonnet-4-6",
        Messages: []talk.Message{{Role: "user", Content: "Hello"}},
    }, func(r talk.Response) error {
        fmt.Print(r.Content)
        return nil
    })

    // Print cost summary
    fmt.Println(agg.Summary().MarkdownTable())
}
```

### Budget Alerts

```go
agg := cost.NewAggregator(
    cost.WithBudget(1.00, func(current, threshold float64) {
        log.Printf("budget exceeded: $%.4f > $%.4f", current, threshold)
    }),
)
```

### Custom Event Store

```go
client := cost.New(
    yourTalkClient,
    rt,
    cost.WithEventStore(myFactEventStore),
)
```

## Built-in Rate Table

| Provider    | Model              | Input ($/M tokens) | Output ($/M tokens) |
|-------------|--------------------|--------------------|---------------------|
| Anthropic   | claude-sonnet-4-6  | $3.00              | $15.00              |
| Anthropic   | claude-haiku-4-5   | $0.80              | $4.00               |
| OpenRouter  | Qwen3.5-122B       | $0.26              | $2.08               |

## YAML Rate Table

Extend or override rates by loading a YAML file:

```go
f, _ := os.Open("my-rates.yaml")
rt.LoadYAML(f)
```

Format — a list of entries:

```yaml
- provider: Anthropic
  model: claude-opus-4-6
  input_per_million: 15.0
  output_per_million: 75.0
- provider: OpenRouter
  model: llama-3-70b
  input_per_million: 0.59
  output_per_million: 0.79
```

`LoadYAML` merges into the existing table, overwriting any entries with the same provider+model key.

## Development

```bash
just test    # run tests with -race
just vet     # go vet
just build   # go build ./...
```
