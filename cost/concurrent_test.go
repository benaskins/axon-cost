package cost

import (
	"context"
	"math"
	"sync"
	"testing"

	talk "github.com/benaskins/axon-talk"
)

// concurrentFixedClient returns a mock LLMClient that always emits the same
// fixed content. It is safe for concurrent use because ChatFn is read-only.
func concurrentFixedClient(content string) *mockLLMClient {
	return &mockLLMClient{
		ChatFn: func(_ context.Context, _ *talk.Request, fn func(talk.Response) error) error {
			return fn(talk.Response{Content: content, Done: true})
		},
	}
}

// streamingClient simulates a provider that delivers chunks from a goroutine,
// as the real openai/anthropic adapters do when parsing SSE. This exercises the
// strings.Builder synchronization in Middleware.Chat.
func streamingClient(chunks []string) *mockLLMClient {
	return &mockLLMClient{
		ChatFn: func(_ context.Context, _ *talk.Request, fn func(talk.Response) error) error {
			errCh := make(chan error, 1)
			go func() {
				for i, c := range chunks {
					if err := fn(talk.Response{Content: c, Done: i == len(chunks)-1}); err != nil {
						errCh <- err
						return
					}
				}
				errCh <- nil
			}()
			return <-errCh
		},
	}
}

func TestMiddlewareAggregator_ConcurrentSafety(t *testing.T) {
	const goroutines = 50
	const inputText = "say hello"
	const outputText = "hello world"

	agg := NewAggregator()
	m := New(
		concurrentFixedClient(outputText),
		DefaultRateTable(),
		WithProvider("Anthropic"),
		WithLabel("concurrent-test"),
		WithAggregator(agg),
	)

	// Compute the expected cost for a single call so we can assert the total.
	singleInputTokens := EstimateTokens(inputText, 0.25)
	singleOutputTokens := EstimateTokens(outputText, 0.25)
	singleRecord, err := Calculate("Anthropic", "claude-sonnet-4-6", singleInputTokens, singleOutputTokens, DefaultRateTable())
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	expectedTotal := float64(goroutines) * singleRecord.TotalCost

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
				{Role: talk.RoleUser, Content: inputText},
			})
			if err := m.Chat(context.Background(), req, func(talk.Response) error { return nil }); err != nil {
				t.Errorf("Chat: %v", err)
			}
		}()
	}
	wg.Wait()

	summary := agg.Summary()
	if summary.TotalCost != expectedTotal {
		// Allow for floating-point rounding noise at the 12th decimal place.
		if math.Abs(summary.TotalCost-expectedTotal) > 1e-10 {
			t.Errorf("TotalCost = %.15f, want %.15f (diff %.2e)",
				summary.TotalCost, expectedTotal, math.Abs(summary.TotalCost-expectedTotal))
		}
	}
	if len(agg.records) != goroutines {
		t.Errorf("record count = %d, want %d", len(agg.records), goroutines)
	}
}

// TestMiddleware_StreamingCallbackRace verifies that Middleware.Chat is safe
// when the inner LLMClient invokes the callback from a separate goroutine,
// as streaming providers (openai, anthropic) do. Run with -race.
func TestMiddleware_StreamingCallbackRace(t *testing.T) {
	const goroutines = 20

	chunks := []string{"hello ", "world ", "from ", "stream"}
	m := New(
		streamingClient(chunks),
		DefaultRateTable(),
		WithProvider("Anthropic"),
		WithAggregator(NewAggregator()),
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			req := talk.NewRequest("claude-sonnet-4-6", []talk.Message{
				{Role: talk.RoleUser, Content: "test"},
			})
			if err := m.Chat(context.Background(), req, func(talk.Response) error { return nil }); err != nil {
				t.Errorf("Chat: %v", err)
			}
		}()
	}
	wg.Wait()
}
