package cost_test

import (
	"context"
	"fmt"

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
