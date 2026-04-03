package cost

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	fact "github.com/benaskins/axon-fact"
	talk "github.com/benaskins/axon-talk"
)

// Middleware wraps a talk.LLMClient to intercept Chat calls and compute
// inference cost. It is safe for concurrent use: all fields are immutable
// after construction; the Aggregator and EventStore manage their own locking.
type Middleware struct {
	inner         talk.LLMClient
	rateTable     *RateTable
	provider      string
	label         string
	aggregator    *Aggregator
	eventStore    fact.EventStore
	tokensPerChar float64
}

// Option configures a Middleware.
type Option func(*Middleware)

// WithProvider sets the provider name used for rate table lookup.
func WithProvider(provider string) Option {
	return func(m *Middleware) { m.provider = provider }
}

// WithLabel sets the label attached to each CostRecord.
func WithLabel(label string) Option {
	return func(m *Middleware) { m.label = label }
}

// WithAggregator arranges for each CostRecord to be recorded in a.
func WithAggregator(a *Aggregator) Option {
	return func(m *Middleware) { m.aggregator = a }
}

// WithEventStore arranges for each CostRecord to be emitted as a fact.Event.
func WithEventStore(es fact.EventStore) Option {
	return func(m *Middleware) { m.eventStore = es }
}

// WithTokensPerChar sets the token estimation ratio (default 0.25).
func WithTokensPerChar(ratio float64) Option {
	return func(m *Middleware) { m.tokensPerChar = ratio }
}

// New constructs a Middleware wrapping inner with the given RateTable and options.
func New(inner talk.LLMClient, rt *RateTable, opts ...Option) *Middleware {
	m := &Middleware{
		inner:         inner,
		rateTable:     rt,
		tokensPerChar: 0.25,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Chat implements talk.LLMClient. It delegates to the inner client, accumulates
// response content, estimates token counts, computes cost, and records the
// result to the optional Aggregator and EventStore.
func (m *Middleware) Chat(ctx context.Context, req *talk.Request, fn func(talk.Response) error) error {
	var sb strings.Builder
	err := m.inner.Chat(ctx, req, func(r talk.Response) error {
		sb.WriteString(r.Content)
		if r.Thinking != "" {
			sb.WriteString(r.Thinking)
		}
		return fn(r)
	})
	if err != nil {
		return err
	}

	inputTokens := EstimateTokens(requestText(req), m.tokensPerChar)
	outputTokens := EstimateTokens(sb.String(), m.tokensPerChar)

	record, calcErr := Calculate(m.provider, req.Model, inputTokens, outputTokens, m.rateTable)
	if calcErr != nil {
		// Cost tracking is best-effort; an unknown rate must not fail the call.
		fmt.Fprintf(os.Stderr, "axon-cost: calculate: %v\n", calcErr)
		return nil
	}
	record.Label = m.label

	if m.aggregator != nil {
		m.aggregator.Record(record)
	}

	if m.eventStore != nil {
		data, jsonErr := json.Marshal(record)
		if jsonErr != nil {
			fmt.Fprintf(os.Stderr, "axon-cost: marshal record: %v\n", jsonErr)
			return nil
		}
		ev := fact.Event{
			ID:         newUUID(),
			Type:       "inference.cost",
			Data:       data,
			OccurredAt: record.Timestamp,
		}
		if appendErr := m.eventStore.Append(ctx, "inference.cost", []fact.Event{ev}); appendErr != nil {
			fmt.Fprintf(os.Stderr, "axon-cost: append event: %v\n", appendErr)
		}
	}

	return nil
}

// newUUID returns a random UUID v4 string using crypto/rand.
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is extraordinarily rare; fall back to empty string
		// rather than panicking — callers treat empty ID as degraded, not fatal.
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// requestText concatenates all message content from req for token estimation.
func requestText(req *talk.Request) string {
	var sb strings.Builder
	for _, msg := range req.Messages {
		sb.WriteString(msg.Content)
	}
	return sb.String()
}
