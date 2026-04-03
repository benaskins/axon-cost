package cost

import (
	"context"
	"encoding/json"
	"testing"

	fact "github.com/benaskins/axon-fact"
	talk "github.com/benaskins/axon-talk"
)

// mockLLMClient is a test double for talk.LLMClient.
type mockLLMClient struct {
	ChatFn func(ctx context.Context, req *talk.Request, fn func(talk.Response) error) error
}

func (m *mockLLMClient) Chat(ctx context.Context, req *talk.Request, fn func(talk.Response) error) error {
	return m.ChatFn(ctx, req, fn)
}

// fixedClient returns a mock that emits a single done response with the given content.
func fixedClient(content string) *mockLLMClient {
	return &mockLLMClient{
		ChatFn: func(_ context.Context, _ *talk.Request, fn func(talk.Response) error) error {
			return fn(talk.Response{Content: content, Done: true})
		},
	}
}

// fakeEventStore captures Append calls for test assertions.
type fakeEventStore struct {
	appended []fact.Event
}

func (f *fakeEventStore) Append(_ context.Context, _ string, events []fact.Event) error {
	f.appended = append(f.appended, events...)
	return nil
}

func (f *fakeEventStore) Load(_ context.Context, _ string) ([]fact.Event, error) {
	return nil, nil
}

func (f *fakeEventStore) LoadFrom(_ context.Context, _ string, _ int64) ([]fact.Event, error) {
	return nil, nil
}

func TestMiddleware_LabelAttached(t *testing.T) {
	agg := &Aggregator{}
	m := New(
		fixedClient("hello world"),
		DefaultRateTable(),
		WithProvider("Anthropic"),
		WithLabel("my-op"),
		WithAggregator(agg),
	)

	req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
		{Role: talk.RoleUser, Content: "say hello"},
	})

	if err := m.Chat(context.Background(), req, func(talk.Response) error { return nil }); err != nil {
		t.Fatalf("Chat: %v", err)
	}

	summary := agg.Summary()
	if _, ok := summary.ByLabel["my-op"]; !ok {
		t.Errorf("expected label 'my-op' in ByLabel, got: %v", summary.ByLabel)
	}
}

func TestMiddleware_AggregatorReceivesRecord(t *testing.T) {
	agg := &Aggregator{}
	m := New(
		fixedClient("response text here"),
		DefaultRateTable(),
		WithProvider("Anthropic"),
		WithLabel("agg-test"),
		WithAggregator(agg),
	)

	req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
		{Role: talk.RoleUser, Content: "test input text"},
	})

	if err := m.Chat(context.Background(), req, func(talk.Response) error { return nil }); err != nil {
		t.Fatalf("Chat: %v", err)
	}

	summary := agg.Summary()
	if summary.TotalCost <= 0 {
		t.Errorf("expected positive TotalCost, got %f", summary.TotalCost)
	}
}

func TestMiddleware_EventStoreAppendCalledWithCorrectType(t *testing.T) {
	store := &fakeEventStore{}
	m := New(
		fixedClient("output content"),
		DefaultRateTable(),
		WithProvider("Anthropic"),
		WithLabel("evtest"),
		WithEventStore(store),
	)

	req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
		{Role: talk.RoleUser, Content: "some input"},
	})

	if err := m.Chat(context.Background(), req, func(talk.Response) error { return nil }); err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if len(store.appended) == 0 {
		t.Fatal("expected at least one event appended to EventStore")
	}
	ev := store.appended[0]
	if ev.Type != "inference.cost" {
		t.Errorf("event type: got %q, want %q", ev.Type, "inference.cost")
	}

	var rec CostRecord
	if err := json.Unmarshal(ev.Data, &rec); err != nil {
		t.Fatalf("unmarshal event data: %v", err)
	}
	if rec.Label != "evtest" {
		t.Errorf("record label: got %q, want %q", rec.Label, "evtest")
	}
}

func TestMiddleware_EventHasNonEmptyID(t *testing.T) {
	store := &fakeEventStore{}
	m := New(
		fixedClient("some output"),
		DefaultRateTable(),
		WithProvider("Anthropic"),
		WithEventStore(store),
	)

	req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
		{Role: talk.RoleUser, Content: "hi"},
	})

	if err := m.Chat(context.Background(), req, func(talk.Response) error { return nil }); err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if len(store.appended) == 0 {
		t.Fatal("expected at least one event appended")
	}
	if store.appended[0].ID == "" {
		t.Error("event ID must be non-empty")
	}
}

func TestMiddleware_EventOccurredAtMatchesRecord(t *testing.T) {
	store := &fakeEventStore{}
	m := New(
		fixedClient("some output"),
		DefaultRateTable(),
		WithProvider("Anthropic"),
		WithEventStore(store),
	)

	req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
		{Role: talk.RoleUser, Content: "hi"},
	})

	if err := m.Chat(context.Background(), req, func(talk.Response) error { return nil }); err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if len(store.appended) == 0 {
		t.Fatal("expected at least one event appended")
	}

	ev := store.appended[0]
	var rec CostRecord
	if err := json.Unmarshal(ev.Data, &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.OccurredAt.IsZero() {
		t.Error("event OccurredAt must not be zero")
	}
	if !ev.OccurredAt.Equal(rec.Timestamp) {
		t.Errorf("OccurredAt %v does not match record Timestamp %v", ev.OccurredAt, rec.Timestamp)
	}
}

// appendErrStore is an EventStore whose Append always returns an error.
type appendErrStore struct{}

func (a *appendErrStore) Append(_ context.Context, _ string, _ []fact.Event) error {
	return errSentinel
}

func (a *appendErrStore) Load(_ context.Context, _ string) ([]fact.Event, error) {
	return nil, nil
}

func (a *appendErrStore) LoadFrom(_ context.Context, _ string, _ int64) ([]fact.Event, error) {
	return nil, nil
}

func TestMiddleware_AppendErrorDoesNotPropagateToChat(t *testing.T) {
	m := New(
		fixedClient("output"),
		DefaultRateTable(),
		WithProvider("Anthropic"),
		WithEventStore(&appendErrStore{}),
	)

	req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
		{Role: talk.RoleUser, Content: "test"},
	})

	if err := m.Chat(context.Background(), req, func(talk.Response) error { return nil }); err != nil {
		t.Errorf("Chat must not return error when Append fails; got: %v", err)
	}
}

func TestMiddleware_PassesThroughInnerError(t *testing.T) {
	inner := &mockLLMClient{
		ChatFn: func(_ context.Context, _ *talk.Request, _ func(talk.Response) error) error {
			return errSentinel
		},
	}
	m := New(inner, DefaultRateTable(), WithProvider("Anthropic"))

	req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
		{Role: talk.RoleUser, Content: "x"},
	})

	err := m.Chat(context.Background(), req, func(talk.Response) error { return nil })
	if err != errSentinel {
		t.Errorf("expected sentinel error, got: %v", err)
	}
}

// errSentinel is a unique error value for testing error propagation.
var errSentinel = &sentinelError{}

type sentinelError struct{}

func (e *sentinelError) Error() string { return "sentinel" }
