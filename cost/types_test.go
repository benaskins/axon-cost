package cost_test

import (
	"testing"
	"time"

	"github.com/benaskins/axon-cost/cost"
)

func TestCostRecordFields(t *testing.T) {
	r := cost.CostRecord{
		Model:         "claude-sonnet-4-6",
		Provider:      "Anthropic",
		InputTokens:   1000,
		OutputTokens:  500,
		InputCost:     0.003,
		OutputCost:    0.0075,
		TotalCost:     0.0105,
		Label:         "test",
		Timestamp:     time.Now(),
	}
	if r.Model != "claude-sonnet-4-6" {
		t.Errorf("unexpected Model: %s", r.Model)
	}
	if r.Provider != "Anthropic" {
		t.Errorf("unexpected Provider: %s", r.Provider)
	}
	if r.InputTokens != 1000 {
		t.Errorf("unexpected InputTokens: %v", r.InputTokens)
	}
	if r.OutputTokens != 500 {
		t.Errorf("unexpected OutputTokens: %v", r.OutputTokens)
	}
}

func TestRateEntryFields(t *testing.T) {
	e := cost.RateEntry{
		InputPerMillion:  3.0,
		OutputPerMillion: 15.0,
	}
	if e.InputPerMillion != 3.0 {
		t.Errorf("unexpected InputPerMillion: %v", e.InputPerMillion)
	}
	if e.OutputPerMillion != 15.0 {
		t.Errorf("unexpected OutputPerMillion: %v", e.OutputPerMillion)
	}
}

func TestRateKeyFields(t *testing.T) {
	k := cost.RateKey{
		Provider: "Anthropic",
		Model:    "claude-sonnet-4-6",
	}
	if k.Provider != "Anthropic" {
		t.Errorf("unexpected Provider: %s", k.Provider)
	}
	if k.Model != "claude-sonnet-4-6" {
		t.Errorf("unexpected Model: %s", k.Model)
	}
}
