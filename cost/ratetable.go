package cost

import (
	"fmt"
	"io"
	"sync"

	"gopkg.in/yaml.v3"
)

// RateTable stores per-model pricing entries, safe for concurrent use.
type RateTable struct {
	mu      sync.RWMutex
	entries map[RateKey]RateEntry
}

// DefaultRateTable returns a RateTable pre-populated with built-in rates.
func DefaultRateTable() *RateTable {
	return &RateTable{
		entries: map[RateKey]RateEntry{
			{Provider: "OpenRouter", Model: "Qwen3.5-122B"}:    {InputPerMillion: 0.26, OutputPerMillion: 2.08},
			{Provider: "Anthropic", Model: "claude-sonnet-4-6"}: {InputPerMillion: 3.0, OutputPerMillion: 15.0},
			{Provider: "Anthropic", Model: "claude-haiku-4-5"}:  {InputPerMillion: 0.80, OutputPerMillion: 4.0},
		},
	}
}

// Lookup returns the RateEntry for the given provider and model.
func (rt *RateTable) Lookup(provider, model string) (RateEntry, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	entry, ok := rt.entries[RateKey{Provider: provider, Model: model}]
	return entry, ok
}

// rateYAMLEntry is the YAML representation of a single rate entry.
type rateYAMLEntry struct {
	Provider         string  `yaml:"provider"`
	Model            string  `yaml:"model"`
	InputPerMillion  float64 `yaml:"input_per_million"`
	OutputPerMillion float64 `yaml:"output_per_million"`
}

// LoadYAML merges rate entries from r into the table, overwriting any
// existing entries for the same provider/model pair.
// Expected format: a YAML list of {provider, model, input_per_million, output_per_million}.
func (rt *RateTable) LoadYAML(r io.Reader) error {
	var rows []rateYAMLEntry
	if err := yaml.NewDecoder(r).Decode(&rows); err != nil {
		return fmt.Errorf("ratetable: decode yaml: %w", err)
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for _, row := range rows {
		rt.entries[RateKey{Provider: row.Provider, Model: row.Model}] = RateEntry{
			InputPerMillion:  row.InputPerMillion,
			OutputPerMillion: row.OutputPerMillion,
		}
	}
	return nil
}
