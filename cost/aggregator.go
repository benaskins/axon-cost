package cost

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Aggregator accumulates CostRecords and computes running totals.
// It is safe for concurrent use.
type Aggregator struct {
	mu      sync.Mutex
	records []CostRecord
}

// Record appends r to the aggregator under lock.
func (a *Aggregator) Record(r CostRecord) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.records = append(a.records, r)
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
