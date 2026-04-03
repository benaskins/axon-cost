package cost

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAggregator_Record_Concurrent(t *testing.T) {
	a := &Aggregator{}
	var wg sync.WaitGroup
	const n = 100
	for i := range n {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			a.Record(CostRecord{
				Model:        "gpt-4",
				Provider:     "openai",
				Label:        "test",
				InputTokens:  i,
				OutputTokens: i * 2,
				TotalCost:    float64(i) * 0.01,
				Timestamp:    time.Now(),
			})
		}()
	}
	wg.Wait()

	s := a.Summary()
	// sum of 0..99 = 4950
	if s.TotalInputTokens != 4950 {
		t.Errorf("TotalInputTokens = %d, want 4950", s.TotalInputTokens)
	}
	if s.TotalOutputTokens != 9900 {
		t.Errorf("TotalOutputTokens = %d, want 9900", s.TotalOutputTokens)
	}
}

func TestAggregator_Summary_Totals(t *testing.T) {
	a := &Aggregator{}
	a.Record(CostRecord{
		Model: "claude-3", Provider: "anthropic", Label: "chat",
		InputTokens: 100, OutputTokens: 200, TotalCost: 0.05,
	})
	a.Record(CostRecord{
		Model: "gpt-4", Provider: "openai", Label: "search",
		InputTokens: 50, OutputTokens: 80, TotalCost: 0.03,
	})
	a.Record(CostRecord{
		Model: "claude-3", Provider: "anthropic", Label: "chat",
		InputTokens: 30, OutputTokens: 60, TotalCost: 0.02,
	})

	s := a.Summary()

	if s.TotalInputTokens != 180 {
		t.Errorf("TotalInputTokens = %d, want 180", s.TotalInputTokens)
	}
	if s.TotalOutputTokens != 340 {
		t.Errorf("TotalOutputTokens = %d, want 340", s.TotalOutputTokens)
	}
	if s.TotalCost != 0.10 {
		t.Errorf("TotalCost = %f, want 0.10", s.TotalCost)
	}
	if s.ByLabel["chat"] != 0.07 {
		t.Errorf("ByLabel[chat] = %f, want 0.07", s.ByLabel["chat"])
	}
	if s.ByLabel["search"] != 0.03 {
		t.Errorf("ByLabel[search] = %f, want 0.03", s.ByLabel["search"])
	}
	if s.ByModel["claude-3"] != 0.07 {
		t.Errorf("ByModel[claude-3] = %f, want 0.07", s.ByModel["claude-3"])
	}
	if s.ByModel["gpt-4"] != 0.03 {
		t.Errorf("ByModel[gpt-4] = %f, want 0.03", s.ByModel["gpt-4"])
	}
}

func TestAggregator_BudgetCallback_Fires(t *testing.T) {
	fired := make(chan struct{}, 1)
	cb := func(current, threshold float64) {
		fired <- struct{}{}
	}

	a := NewAggregator(WithBudget(0.05, cb))
	a.Record(CostRecord{
		Model: "gpt-4", Provider: "openai", Label: "test",
		InputTokens: 100, OutputTokens: 100, TotalCost: 0.06,
	})

	select {
	case <-fired:
		// expected
	case <-time.After(time.Second):
		t.Error("budget callback did not fire within 1s")
	}
}

func TestAggregator_BudgetCallback_DoesNotFireWhenUnder(t *testing.T) {
	fired := make(chan struct{}, 1)
	cb := func(current, threshold float64) {
		fired <- struct{}{}
	}

	a := NewAggregator(WithBudget(0.10, cb))
	a.Record(CostRecord{
		Model: "gpt-4", Provider: "openai", Label: "test",
		InputTokens: 100, OutputTokens: 100, TotalCost: 0.05,
	})

	select {
	case <-fired:
		t.Error("budget callback fired but should not have")
	case <-time.After(50 * time.Millisecond):
		// expected: no fire
	}
}

func TestAggregator_BudgetCallback_FiresOnce(t *testing.T) {
	var count int
	var mu sync.Mutex
	cb := func(current, threshold float64) {
		mu.Lock()
		count++
		mu.Unlock()
	}

	a := NewAggregator(WithBudget(0.05, cb))
	// Two records each exceeding the threshold
	a.Record(CostRecord{
		Model: "gpt-4", Provider: "openai", Label: "test",
		InputTokens: 100, OutputTokens: 100, TotalCost: 0.06,
	})
	a.Record(CostRecord{
		Model: "gpt-4", Provider: "openai", Label: "test",
		InputTokens: 100, OutputTokens: 100, TotalCost: 0.06,
	})

	// Give goroutines time to run
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	got := count
	mu.Unlock()
	if got != 1 {
		t.Errorf("budget callback fired %d times, want 1", got)
	}
}

func TestCostSummary_MarkdownTable(t *testing.T) {
	s := CostSummary{
		TotalInputTokens:  180,
		TotalOutputTokens: 340,
		TotalCost:         0.10,
		ByLabel:           map[string]float64{"chat": 0.07, "search": 0.03},
		ByModel:           map[string]float64{"claude-3": 0.07, "gpt-4": 0.03},
	}

	md := s.MarkdownTable()

	// Must have a header row
	if !strings.Contains(md, "| Label/Model") {
		t.Errorf("missing header row, got:\n%s", md)
	}
	// Must have separator row
	if !strings.Contains(md, "|---") {
		t.Errorf("missing separator row, got:\n%s", md)
	}
	// Must contain label rows
	if !strings.Contains(md, "chat") {
		t.Errorf("missing 'chat' row, got:\n%s", md)
	}
	if !strings.Contains(md, "search") {
		t.Errorf("missing 'search' row, got:\n%s", md)
	}
	// Must contain model rows
	if !strings.Contains(md, "claude-3") {
		t.Errorf("missing 'claude-3' row, got:\n%s", md)
	}
	if !strings.Contains(md, "gpt-4") {
		t.Errorf("missing 'gpt-4' row, got:\n%s", md)
	}
	// Must contain cost values
	if !strings.Contains(md, "0.10") {
		t.Errorf("missing total cost in table, got:\n%s", md)
	}
}
