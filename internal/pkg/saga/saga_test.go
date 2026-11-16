package saga

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSaga_Success(t *testing.T) {
	var executed []string
	var compensated []string

	s := New()
	s.AddStep("create_order",
		func(ctx context.Context) error { executed = append(executed, "order"); return nil },
		func(ctx context.Context) error { compensated = append(compensated, "order"); return nil },
	)
	s.AddStep("reserve_inventory",
		func(ctx context.Context) error { executed = append(executed, "inventory"); return nil },
		func(ctx context.Context) error { compensated = append(compensated, "inventory"); return nil },
	)
	s.AddStep("charge_payment",
		func(ctx context.Context) error { executed = append(executed, "payment"); return nil },
		func(ctx context.Context) error { compensated = append(compensated, "payment"); return nil },
	)

	err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected saga error: %v", err)
	}

	if len(executed) != 3 || executed[0] != "order" || executed[1] != "inventory" || executed[2] != "payment" {
		t.Fatalf("unexpected execution sequence: %v", executed)
	}
	if len(compensated) != 0 {
		t.Fatalf("did not expect compensations on success")
	}
}

func TestSaga_BackwardCompensation(t *testing.T) {
	var executed []string
	var compensated []string
	paymentErr := errors.New("insufficient balance")

	s := New()
	s.AddStep("create_order",
		func(ctx context.Context) error { executed = append(executed, "order"); return nil },
		func(ctx context.Context) error { compensated = append(compensated, "order"); return nil },
	)
	s.AddStep("reserve_inventory",
		func(ctx context.Context) error { executed = append(executed, "inventory"); return nil },
		func(ctx context.Context) error { compensated = append(compensated, "inventory"); return nil },
	)
	s.AddStep("charge_payment",
		func(ctx context.Context) error { return paymentErr }, // fails!
		func(ctx context.Context) error { compensated = append(compensated, "payment"); return nil },
	)

	err := s.Run(context.Background())
	if !errors.Is(err, ErrSagaFailed) {
		t.Fatalf("expected ErrSagaFailed, got: %v", err)
	}

	// Forward: order, inventory
	if len(executed) != 2 {
		t.Fatalf("expected 2 forward steps, got %d", len(executed))
	}

	// Backward: inventory first, then order (reverse order!)
	if len(compensated) != 2 || compensated[0] != "inventory" || compensated[1] != "order" {
		t.Fatalf("expected reverse compensation [inventory, order], got: %v", compensated)
	}
}

func TestSaga_PanicRecovery(t *testing.T) {
	var compensated []string

	s := New()
	s.AddStep("step1",
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { compensated = append(compensated, "step1"); return nil },
	)
	s.AddStep("step2",
		func(ctx context.Context) error { panic("fatal crash") },
		func(ctx context.Context) error { return nil },
	)

	err := s.Run(context.Background())
	if !errors.Is(err, ErrSagaFailed) {
		t.Fatalf("expected ErrSagaFailed, got: %v", err)
	}
	if len(compensated) != 1 || compensated[0] != "step1" {
		t.Fatalf("expected step1 compensated after step2 panic")
	}
}

func TestSaga_CompensationFailure(t *testing.T) {
	s := New()
	s.AddStep("step1",
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return errors.New("cannot refund money") },
	)
	s.AddStep("step2",
		func(ctx context.Context) error { return errors.New("upstream failure") },
		func(ctx context.Context) error { return nil },
	)

	err := s.Run(context.Background())
	if !errors.Is(err, ErrCompensationFailed) {
		t.Fatalf("expected ErrCompensationFailed, got: %v", err)
	}
	if !strings.Contains(err.Error(), "cannot refund money") {
		t.Fatalf("missing compensation error detail: %s", err.Error())
	}
}

func TestSaga_Empty(t *testing.T) {
	s := New()
	err := s.Run(context.Background())
	if !errors.Is(err, ErrEmptySaga) {
		t.Fatalf("expected ErrEmptySaga, got: %v", err)
	}
}
