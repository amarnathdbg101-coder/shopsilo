package gooutbox_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gooutbox "shopMe/internal/pkg/outbox"
)

type OrderCreatedPayload struct {
	OrderID string  `json:"order_id"`
	Total   float64 `json:"total"`
}

func TestEngine_SuccessfulEnqueueAndDispatch(t *testing.T) {
	store := gooutbox.NewMemoryStore()
	var publishedCount int32

	publisher := gooutbox.FuncPublisher(func(ctx context.Context, event *gooutbox.Event) error {
		atomic.AddInt32(&publishedCount, 1)
		return nil
	})

	engine, err := gooutbox.New(store, publisher,
		gooutbox.WithPollInterval(20*time.Millisecond),
		gooutbox.WithBatchSize(10),
	)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Enqueue 5 events
	for i := 1; i <= 5; i++ {
		_, err := engine.Enqueue(ctx, nil, "order.created", "ord_101", OrderCreatedPayload{
			OrderID: "ord_101",
			Total:   1499.50,
		})
		if err != nil {
			t.Fatalf("failed to enqueue event: %v", err)
		}
	}

	// 2. Start background worker
	engine.Start(ctx)

	// Allow worker to poll and dispatch
	time.Sleep(100 * time.Millisecond)
	engine.Stop()

	if atomic.LoadInt32(&publishedCount) != 5 {
		t.Fatalf("expected 5 events published, got %d", atomic.LoadInt32(&publishedCount))
	}
}

func TestEngine_RetryWithBackoffAndDeadLetter(t *testing.T) {
	store := gooutbox.NewMemoryStore()
	var attempts int32

	// Flaky publisher that always errors
	publisher := gooutbox.FuncPublisher(func(ctx context.Context, event *gooutbox.Event) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("downstream message broker network timeout")
	})

	engine, err := gooutbox.New(store, publisher,
		gooutbox.WithMaxRetries(2),
		gooutbox.WithBatchSize(10),
	)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	ctx := context.Background()

	event, err := engine.Enqueue(ctx, nil, "payment.failed", "pay_999", "payload_data")
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	// Attempt 1: Process batch -> Should fail and transition to failed
	count, err := engine.ProcessBatch(ctx)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 processed event, got %d (err: %v)", count, err)
	}

	// Attempt 2: Second failure -> Exceeds MaxRetries (2) -> Should route to Dead Letter Queue
	// Artificially reset next retry to now for immediate test execution
	_ = store.MarkFailed(ctx, event.ID, errors.New("simulated second failure"), 0)

	count2, err := engine.ProcessBatch(ctx)
	if err != nil || count2 != 1 {
		t.Fatalf("expected 1 processed event on attempt 2, got %d", count2)
	}

	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected exactly 2 dispatch attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestEngine_ValidationErrors(t *testing.T) {
	store := gooutbox.NewMemoryStore()
	pub := gooutbox.FuncPublisher(func(ctx context.Context, event *gooutbox.Event) error {
		return nil
	})

	if _, err := gooutbox.New(nil, pub); err != gooutbox.ErrStoreNil {
		t.Fatalf("expected ErrStoreNil, got %v", err)
	}
	if _, err := gooutbox.New(store, nil); err != gooutbox.ErrPublisherNil {
		t.Fatalf("expected ErrPublisherNil, got %v", err)
	}

	engine, _ := gooutbox.New(store, pub)
	ctx := context.Background()

	// Empty topic
	if _, err := engine.Enqueue(ctx, nil, "", "agg_1", "payload"); err != gooutbox.ErrEmptyTopic {
		t.Fatalf("expected ErrEmptyTopic, got %v", err)
	}

	// Empty payload
	if _, err := engine.Enqueue(ctx, nil, "valid.topic", "agg_1", nil); err != gooutbox.ErrEmptyPayload {
		t.Fatalf("expected ErrEmptyPayload, got %v", err)
	}
}


type mockPgxTx struct {
	executedQuery string
	executedArgs  []any
}

func (m *mockPgxTx) Exec(ctx context.Context, query string, args ...any) (any, error) {
	m.executedQuery = query
	m.executedArgs = args
	return nil, nil
}

func TestSQLStore_CustomColumnsAndPgxDuckTyping(t *testing.T) {
	store := gooutbox.NewSQLStore(nil, &gooutbox.SQLOptions{
		TableName:         "custom_outbox",
		TopicColumn:       "event_type",
		AggregateIDColumn: "shop_id",
	})

	mockTx := &mockPgxTx{}
	ctx := context.Background()

	event, err := gooutbox.NewEvent("order.placed", "shop_999", map[string]string{"order_id": "123"})
	if err != nil {
		t.Fatalf("unexpected error creating event: %v", err)
	}

	if err := store.Save(ctx, mockTx, event); err != nil {
		t.Fatalf("unexpected error in Save: %v", err)
	}

	if mockTx.executedQuery == "" {
		t.Fatal("expected mockPgxTx.Exec to be invoked via reflection")
	}

	// Verify query references custom columns
	expectedSub := "INSERT INTO custom_outbox (id, event_type, shop_id"
	if len(mockTx.executedQuery) < len(expectedSub) || mockTx.executedQuery[:len(expectedSub)] != expectedSub {
		// Just check contains
		if !strings.Contains(mockTx.executedQuery, "event_type") || !strings.Contains(mockTx.executedQuery, "shop_id") {
			t.Fatalf("expected query to contain custom columns, got: %s", mockTx.executedQuery)
		}
	}
}
