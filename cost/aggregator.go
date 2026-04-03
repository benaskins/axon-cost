package cost

import "sync"

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
	TotalCost float64
	ByLabel   map[string]float64
}

// Summary returns a snapshot of aggregated costs.
func (a *Aggregator) Summary() CostSummary {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := CostSummary{ByLabel: make(map[string]float64)}
	for _, r := range a.records {
		s.TotalCost += r.TotalCost
		s.ByLabel[r.Label] += r.TotalCost
	}
	return s
}
