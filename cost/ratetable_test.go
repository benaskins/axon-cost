package cost_test

import (
	"strings"
	"testing"

	"github.com/benaskins/axon-cost/cost"
)

func TestDefaultRateTableBuiltins(t *testing.T) {
	rt := cost.DefaultRateTable()

	cases := []struct {
		provider string
		model    string
		in       float64
		out      float64
	}{
		{"OpenRouter", "Qwen3.5-122B", 0.26, 2.08},
		{"Anthropic", "claude-sonnet-4-6", 3.0, 15.0},
		{"Anthropic", "claude-haiku-4-5", 0.80, 4.0},
	}

	for _, c := range cases {
		entry, ok := rt.Lookup(c.provider, c.model)
		if !ok {
			t.Errorf("expected built-in entry for %s/%s", c.provider, c.model)
			continue
		}
		if entry.InputPerMillion != c.in {
			t.Errorf("%s/%s: InputPerMillion: want %v, got %v", c.provider, c.model, c.in, entry.InputPerMillion)
		}
		if entry.OutputPerMillion != c.out {
			t.Errorf("%s/%s: OutputPerMillion: want %v, got %v", c.provider, c.model, c.out, entry.OutputPerMillion)
		}
	}
}

func TestLookupUnknownReturnsFalse(t *testing.T) {
	rt := cost.DefaultRateTable()
	_, ok := rt.Lookup("Unknown", "nonexistent-model")
	if ok {
		t.Error("expected false for unknown model")
	}
}

func TestLoadYAMLMerge(t *testing.T) {
	rt := cost.DefaultRateTable()

	yaml := `
- provider: TestProvider
  model: test-model-1
  input_per_million: 1.5
  output_per_million: 3.0
- provider: TestProvider
  model: test-model-2
  input_per_million: 2.0
  output_per_million: 6.0
`
	if err := rt.LoadYAML(strings.NewReader(yaml)); err != nil {
		t.Fatalf("LoadYAML error: %v", err)
	}

	entry, ok := rt.Lookup("TestProvider", "test-model-1")
	if !ok {
		t.Fatal("expected merged entry for TestProvider/test-model-1")
	}
	if entry.InputPerMillion != 1.5 {
		t.Errorf("InputPerMillion: want 1.5, got %v", entry.InputPerMillion)
	}
	if entry.OutputPerMillion != 3.0 {
		t.Errorf("OutputPerMillion: want 3.0, got %v", entry.OutputPerMillion)
	}

	// Built-in entries must survive the merge.
	_, ok = rt.Lookup("Anthropic", "claude-sonnet-4-6")
	if !ok {
		t.Error("built-in entry missing after YAML merge")
	}
}

func TestLoadYAMLOverridesBuiltin(t *testing.T) {
	rt := cost.DefaultRateTable()

	yaml := `
- provider: Anthropic
  model: claude-sonnet-4-6
  input_per_million: 99.0
  output_per_million: 199.0
`
	if err := rt.LoadYAML(strings.NewReader(yaml)); err != nil {
		t.Fatalf("LoadYAML error: %v", err)
	}

	entry, ok := rt.Lookup("Anthropic", "claude-sonnet-4-6")
	if !ok {
		t.Fatal("expected entry after YAML override")
	}
	if entry.InputPerMillion != 99.0 {
		t.Errorf("InputPerMillion: want 99.0, got %v", entry.InputPerMillion)
	}
	if entry.OutputPerMillion != 199.0 {
		t.Errorf("OutputPerMillion: want 199.0, got %v", entry.OutputPerMillion)
	}
}

func TestLoadYAMLInvalidInput(t *testing.T) {
	rt := cost.DefaultRateTable()
	err := rt.LoadYAML(strings.NewReader("not: valid: yaml: list"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
