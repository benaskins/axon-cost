package cost

import "time"

// CostRecord holds the computed cost for a single LLM inference call.
type CostRecord struct {
	Model        string
	Provider     string
	InputTokens  int
	OutputTokens int
	InputCost    float64
	OutputCost   float64
	TotalCost    float64
	Label        string
	Timestamp    time.Time
}

// RateEntry holds per-million-token pricing for a model.
type RateEntry struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// RateKey identifies a specific model from a specific provider.
type RateKey struct {
	Provider string
	Model    string
}
