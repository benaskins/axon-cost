package cost

import (
	"math"
	"testing"
	"time"
)

const floatEpsilon = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < floatEpsilon
}

func TestCalculate(t *testing.T) {
	rt := &RateTable{
		entries: map[RateKey]RateEntry{
			{Provider: "Anthropic", Model: "claude-sonnet-4-6"}: {InputPerMillion: 3.0, OutputPerMillion: 15.0},
		},
	}

	t.Run("known rate produces correct costs", func(t *testing.T) {
		before := time.Now()
		rec, err := Calculate("Anthropic", "claude-sonnet-4-6", 1_000_000, 500_000, rt)
		after := time.Now()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rec.Provider != "Anthropic" {
			t.Errorf("provider: got %q, want %q", rec.Provider, "Anthropic")
		}
		if rec.Model != "claude-sonnet-4-6" {
			t.Errorf("model: got %q, want %q", rec.Model, "claude-sonnet-4-6")
		}
		if rec.InputTokens != 1_000_000 {
			t.Errorf("input tokens: got %d, want %d", rec.InputTokens, 1_000_000)
		}
		if rec.OutputTokens != 500_000 {
			t.Errorf("output tokens: got %d, want %d", rec.OutputTokens, 500_000)
		}
		// 1_000_000 / 1_000_000 * 3.0 = 3.0
		if rec.InputCost != 3.0 {
			t.Errorf("input cost: got %f, want %f", rec.InputCost, 3.0)
		}
		// 500_000 / 1_000_000 * 15.0 = 7.5
		if rec.OutputCost != 7.5 {
			t.Errorf("output cost: got %f, want %f", rec.OutputCost, 7.5)
		}
		// 3.0 + 7.5 = 10.5
		if rec.TotalCost != 10.5 {
			t.Errorf("total cost: got %f, want %f", rec.TotalCost, 10.5)
		}
		if rec.Timestamp.Before(before) || rec.Timestamp.After(after) {
			t.Errorf("timestamp %v not between %v and %v", rec.Timestamp, before, after)
		}
	})

	t.Run("fractional tokens round correctly", func(t *testing.T) {
		// 100 input tokens: 100 / 1_000_000 * 3.0 = 0.0003
		rec, err := Calculate("Anthropic", "claude-sonnet-4-6", 100, 0, rt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := 100.0 / 1_000_000.0 * 3.0
		if !approxEqual(rec.InputCost, want) {
			t.Errorf("input cost: got %v, want %v", rec.InputCost, want)
		}
		if rec.OutputCost != 0.0 {
			t.Errorf("output cost: got %v, want 0", rec.OutputCost)
		}
		if !approxEqual(rec.TotalCost, want) {
			t.Errorf("total cost: got %v, want %v", rec.TotalCost, want)
		}
	})

	t.Run("unknown rate returns error", func(t *testing.T) {
		_, err := Calculate("Unknown", "no-such-model", 100, 100, rt)
		if err == nil {
			t.Fatal("expected error for unknown rate, got nil")
		}
	})
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		tokensPerChar float64
		want          int
	}{
		{"empty string", "", 0.25, 0},
		{"four chars at default ratio", "abcd", 0.25, 1},
		{"exact division", "abcdefgh", 0.25, 2},
		{"rounds down", "abc", 0.25, 0},
		{"custom ratio", "hello", 1.0, 5},
		{"custom ratio fractional", "hello", 0.5, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input, tt.tokensPerChar)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q, %v) = %d, want %d", tt.input, tt.tokensPerChar, got, tt.want)
			}
		})
	}
}
