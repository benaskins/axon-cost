package cost

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// AggregatorOption configures an Aggregator.
type AggregatorOption func(*Aggregator)

// WithBudget sets a cost threshold. When the running TotalCost exceeds
// threshold, cb is called once in a goroutine (non-blocking). It never
// re-fires; create a new Aggregator to reset.
func WithBudget(threshold float64, cb func(current, threshold float64)) AggregatorOption {
	return func(a *Aggregator) {
		a.budgetThreshold = threshold
		a.budgetCallback = cb
	}
}

// NewAggregator returns an Aggregator configured with the given options.
func NewAggregator(opts ...AggregatorOption) *Aggregator {
	a := &Aggregator{}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Aggregator accumulates CostRecords and computes running totals.
// It is safe for concurrent use.
type Aggregator struct {
	mu              sync.Mutex
	records         []CostRecord
	budgetThreshold float64
	budgetCallback  func(current, threshold float64)
	budgetFired     bool
}

// Record appends r to the aggregator under lock.
// If a budget threshold is set and total cost exceeds it for the first time,
// the callback is fired once in a goroutine.
func (a *Aggregator) Record(r CostRecord) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.records = append(a.records, r)
	if a.budgetCallback != nil && !a.budgetFired && a.budgetThreshold > 0 {
		var total float64
		for _, rec := range a.records {
			total += rec.TotalCost
		}
		if total > a.budgetThreshold {
			a.budgetFired = true
			cb := a.budgetCallback
			threshold := a.budgetThreshold
			go cb(total, threshold)
		}
	}
}

// CostSummary holds aggregated cost data across recorded calls.
type CostSummary struct {
	TotalInputTokens  int
	TotalOutputTokens int
	TotalCost         float64
	ByLabel           map[string]float64
	ByModel           map[string]float64
}

// Summary returns a snapshot of aggregated costs.
func (a *Aggregator) Summary() CostSummary {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := CostSummary{
		ByLabel: make(map[string]float64),
		ByModel: make(map[string]float64),
	}
	for _, r := range a.records {
		s.TotalInputTokens += r.InputTokens
		s.TotalOutputTokens += r.OutputTokens
		s.TotalCost += r.TotalCost
		s.ByLabel[r.Label] += r.TotalCost
		s.ByModel[r.Model] += r.TotalCost
	}
	return s
}

// MarkdownTable renders a markdown table with columns:
// Label/Model, Input Tokens, Output Tokens, Cost (USD).
func (cs CostSummary) MarkdownTable() string {
	var sb strings.Builder

	sb.WriteString("| Label/Model | Input Tokens | Output Tokens | Cost (USD) |\n")
	sb.WriteString("|---|---|---|---|\n")

	// Labels section
	labels := make([]string, 0, len(cs.ByLabel))
	for l := range cs.ByLabel {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	for _, l := range labels {
		fmt.Fprintf(&sb, "| %s | - | - | %.4f |\n", l, cs.ByLabel[l])
	}

	// Models section
	models := make([]string, 0, len(cs.ByModel))
	for m := range cs.ByModel {
		models = append(models, m)
	}
	sort.Strings(models)
	for _, m := range models {
		fmt.Fprintf(&sb, "| %s | - | - | %.4f |\n", m, cs.ByModel[m])
	}

	// Totals row
	fmt.Fprintf(&sb, "| **Total** | %d | %d | %.4f |\n",
		cs.TotalInputTokens, cs.TotalOutputTokens, cs.TotalCost)

	return sb.String()
}
