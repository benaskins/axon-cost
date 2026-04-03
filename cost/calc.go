package cost

import (
	"fmt"
	"time"
)

// Calculate computes the cost of an LLM inference call using the given rate table.
// Returns an error if no rate entry is found for the provider/model pair.
func Calculate(provider, model string, inputTokens, outputTokens int, rt *RateTable) (CostRecord, error) {
	entry, ok := rt.Lookup(provider, model)
	if !ok {
		return CostRecord{}, fmt.Errorf("calc: no rate found for provider %q model %q", provider, model)
	}
	inputCost := float64(inputTokens) / 1_000_000.0 * entry.InputPerMillion
	outputCost := float64(outputTokens) / 1_000_000.0 * entry.OutputPerMillion
	return CostRecord{
		Provider:     provider,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		InputCost:    inputCost,
		OutputCost:   outputCost,
		TotalCost:    inputCost + outputCost,
		Timestamp:    time.Now(),
	}, nil
}

// EstimateTokens estimates the number of tokens in s using the given ratio.
// A tokensPerChar of 0.25 reflects the common approximation of ~4 characters per token.
func EstimateTokens(s string, tokensPerChar float64) int {
	return int(float64(len(s)) * tokensPerChar)
}
