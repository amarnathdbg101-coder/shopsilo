package goshutdown

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShutdown_PhasedExecutionOrder(t *testing.T) {
	c := New(5 * time.Second)

	var executionOrder []string

	// Register out of order
	c.RegisterStorage("database_close", func(ctx context.Context) error {
		executionOrder = append(executionOrder, "phase_3_db")
		return nil
	})

	c.RegisterHTTP("http_server_stop", func(ctx context.Context) error {
		executionOrder = append(executionOrder, "phase_1_http")
		return nil
	})

	c.RegisterWorker("worker_pool_drain", func(ctx context.Context) error {
		executionOrder = append(executionOrder, "phase_2_worker")
		return nil
	})

	// Trigger manual shutdown
	err := c.Trigger(context.Background())
	if err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 hooks executed, got %d", len(executionOrder))
	}

	// Must be: phase 1 -> phase 2 -> phase 3
	if executionOrder[0] != "phase_1_http" || executionOrder[1] != "phase_2_worker" || executionOrder[2] != "phase_3_db" {
		t.Fatalf("phased order violation: %+v", executionOrder)
	}
}

func TestShutdown_ContextTimeout(t *testing.T) {
	c := New(50 * time.Millisecond)

	c.Register(1, "hanging_job", func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := c.Trigger(ctx)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestShutdown_CollectsErrors(t *testing.T) {
	c := New(2 * time.Second)

	c.Register(1, "failing_service", func(ctx context.Context) error {
		return errors.New("connection failed to close")
	})

	err := c.Trigger(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
