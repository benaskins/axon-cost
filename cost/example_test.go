package cost_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/benaskins/axon-cost/cost"
	talk "github.com/benaskins/axon-talk"
)

// stubClient is a minimal talk.LLMClient for demonstration.
type stubClient struct{}

func (stubClient) Chat(context.Context, *talk.Request, func(talk.Response) error) error {
	return nil
}

func ExampleNewAggregator() {
	agg := cost.NewAggregator()

	agg.Record(cost.CostRecord{
		Model:        "claude-sonnet-4-20250514",
		Provider:     "anthropic",
		InputTokens:  1000,
		OutputTokens: 500,
		TotalCost:    0.0045,
		Label:        "chat",
	})

	s := agg.Summary()
	fmt.Printf("total: $%.4f  input_tokens: %d  output_tokens: %d\n",
		s.TotalCost, s.TotalInputTokens, s.TotalOutputTokens)
	// Output: total: $0.0045  input_tokens: 1000  output_tokens: 500
}

func ExampleNew() {
	agg := cost.NewAggregator()
	rt := cost.DefaultRateTable()

	client := cost.New(
		stubClient{},
		rt,
		cost.WithProvider("anthropic"),
		cost.WithLabel("planning"),
		cost.WithAggregator(agg),
	)

	// client satisfies talk.LLMClient and tracks cost on every Chat call.
	_ = client
	fmt.Println("ok")
	// Output: ok
}

// Calculate computes the cost of an inference call from token counts
// and a rate table. The returned CostRecord breaks cost into input and
// output components.
func ExampleCalculate() {
	rt := cost.DefaultRateTable()

	rec, err := cost.Calculate("Anthropic", "claude-sonnet-4-6", 10000, 2000, rt)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("input:  $%.4f\n", rec.InputCost)
	fmt.Printf("output: $%.4f\n", rec.OutputCost)
	fmt.Printf("total:  $%.4f\n", rec.TotalCost)
	// Output:
	// input:  $0.0300
	// output: $0.0300
	// total:  $0.0600
}

// Calculate returns an error when the provider/model pair has no rate entry.
func ExampleCalculate_unknownModel() {
	rt := cost.DefaultRateTable()

	_, err := cost.Calculate("unknown", "no-such-model", 100, 50, rt)
	fmt.Println(err)
	// Output: calc: no rate found for provider "unknown" model "no-such-model"
}

// EstimateTokens approximates the number of tokens in a string using a
// characters-per-token ratio.
func ExampleEstimateTokens() {
	text := "Hello, world!" // 13 characters
	tokens := cost.EstimateTokens(text, 0.25)
	fmt.Println(tokens, "tokens")
	// Output: 3 tokens
}

// DefaultRateTable ships with built-in pricing for common models.
// Use Lookup to retrieve a rate entry by provider and model name.
func ExampleDefaultRateTable() {
	rt := cost.DefaultRateTable()

	entry, ok := rt.Lookup("Anthropic", "claude-haiku-4-5")
	fmt.Println("found:", ok)
	fmt.Printf("input:  $%.2f/M tokens\n", entry.InputPerMillion)
	fmt.Printf("output: $%.2f/M tokens\n", entry.OutputPerMillion)
	// Output:
	// found: true
	// input:  $0.80/M tokens
	// output: $4.00/M tokens
}

// LoadYAML merges additional rate entries into a RateTable from a YAML
// source. This lets you override built-in rates or add new models.
func ExampleRateTable_LoadYAML() {
	rt := cost.DefaultRateTable()

	yaml := `
- provider: Local
  model: llama-3.3-70b
  input_per_million: 0.0
  output_per_million: 0.0
`
	if err := rt.LoadYAML(strings.NewReader(yaml)); err != nil {
		fmt.Println(err)
		return
	}

	entry, ok := rt.Lookup("Local", "llama-3.3-70b")
	fmt.Println("found:", ok)
	fmt.Printf("input:  $%.2f/M tokens\n", entry.InputPerMillion)
	// Output:
	// found: true
	// input:  $0.00/M tokens
}

// An Aggregator with multiple records produces a Summary broken down
// by label and model.
func ExampleAggregator_Summary() {
	agg := cost.NewAggregator()

	agg.Record(cost.CostRecord{
		Model: "claude-sonnet-4-6", InputTokens: 500, OutputTokens: 200,
		TotalCost: 0.0045, Label: "planning",
	})
	agg.Record(cost.CostRecord{
		Model: "claude-haiku-4-5", InputTokens: 1000, OutputTokens: 300,
		TotalCost: 0.0020, Label: "coding",
	})
	agg.Record(cost.CostRecord{
		Model: "claude-sonnet-4-6", InputTokens: 800, OutputTokens: 400,
		TotalCost: 0.0055, Label: "planning",
	})

	s := agg.Summary()
	fmt.Printf("total_cost: $%.4f\n", s.TotalCost)
	fmt.Printf("input_tokens: %d\n", s.TotalInputTokens)
	fmt.Printf("output_tokens: %d\n", s.TotalOutputTokens)
	fmt.Printf("planning: $%.4f\n", s.ByLabel["planning"])
	fmt.Printf("coding: $%.4f\n", s.ByLabel["coding"])
	fmt.Printf("sonnet: $%.4f\n", s.ByModel["claude-sonnet-4-6"])
	fmt.Printf("haiku: $%.4f\n", s.ByModel["claude-haiku-4-5"])
	// Output:
	// total_cost: $0.0120
	// input_tokens: 2300
	// output_tokens: 900
	// planning: $0.0100
	// coding: $0.0020
	// sonnet: $0.0100
	// haiku: $0.0020
}

// MarkdownTable renders a CostSummary as a markdown table suitable for
// CLI output or documentation.
func ExampleCostSummary_MarkdownTable() {
	agg := cost.NewAggregator()
	agg.Record(cost.CostRecord{
		Model: "claude-sonnet-4-6", InputTokens: 1000, OutputTokens: 500,
		TotalCost: 0.0105, Label: "chat",
	})

	table := agg.Summary().MarkdownTable()
	fmt.Print(table)
	// Output:
	// | Label/Model | Input Tokens | Output Tokens | Cost (USD) |
	// |---|---|---|---|
	// | chat | - | - | 0.0105 |
	// | claude-sonnet-4-6 | - | - | 0.0105 |
	// | **Total** | 1000 | 500 | 0.0105 |
}
