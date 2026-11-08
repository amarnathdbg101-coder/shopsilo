package services

import (
	"context"
	"errors"
	"testing"

	saga "github.com/amarnathdbg101-coder/go/go-saga"
)

func TestOrderSaga_SuccessfulOrchestration(t *testing.T) {
	var stepOrder []string

	coord := saga.New()
	coord.AddStep(
		"ValidateCart",
		func(ctx context.Context) error {
			stepOrder = append(stepOrder, "validate")
			return nil
		},
		func(ctx context.Context) error {
			stepOrder = append(stepOrder, "compensate_validate")
			return nil
		},
	)
	coord.AddStep(
		"ReserveInventory",
		func(ctx context.Context) error {
			stepOrder = append(stepOrder, "reserve")
			return nil
		},
		func(ctx context.Context) error {
			stepOrder = append(stepOrder, "compensate_reserve")
			return nil
		},
	)
	coord.AddStep(
		"ProcessPayment",
		func(ctx context.Context) error {
			stepOrder = append(stepOrder, "payment")
			return nil
		},
		func(ctx context.Context) error {
			stepOrder = append(stepOrder, "compensate_payment")
			return nil
		},
	)
	coord.AddStep(
		"NotifyCustomer",
		func(ctx context.Context) error {
			stepOrder = append(stepOrder, "notify")
			return nil
		},
		nil,
	)

	err := coord.Run(context.Background())
	if err != nil {
		t.Fatalf("expected saga to succeed, got: %v", err)
	}

	expected := []string{"validate", "reserve", "payment", "notify"}
	if len(stepOrder) != len(expected) {
		t.Fatalf("step count mismatch: got %v, want %v", stepOrder, expected)
	}
	for i := range expected {
		if stepOrder[i] != expected[i] {
			t.Errorf("step[%d] = %s, want %s", i, stepOrder[i], expected[i])
		}
	}
}

func TestOrderSaga_ReverseCompensationOnFailure(t *testing.T) {
	var stepLog []string

	coord := saga.New()
	// Step 1: Validate
	coord.AddStep(
		"ValidateOrder",
		func(ctx context.Context) error {
			stepLog = append(stepLog, "exec_validate")
			return nil
		},
		func(ctx context.Context) error {
			stepLog = append(stepLog, "comp_validate")
			return nil
		},
	)
	// Step 2: Reserve Inventory
	coord.AddStep(
		"ReserveStock",
		func(ctx context.Context) error {
			stepLog = append(stepLog, "exec_reserve")
			return nil
		},
		func(ctx context.Context) error {
			stepLog = append(stepLog, "comp_reserve_restock")
			return nil
		},
	)
	// Step 3: Ledger Debit
	coord.AddStep(
		"CustomerKhataDebit",
		func(ctx context.Context) error {
			stepLog = append(stepLog, "exec_khata_debit")
			return nil
		},
		func(ctx context.Context) error {
			stepLog = append(stepLog, "comp_khata_credit_reversal")
			return nil
		},
	)
	// Step 4: External Notification / Cash Drawer Fails!
	coord.AddStep(
		"ExternalPaymentGateway",
		func(ctx context.Context) error {
			stepLog = append(stepLog, "exec_payment_failed")
			return errors.New("gateway network timeout")
		},
		func(ctx context.Context) error {
			stepLog = append(stepLog, "comp_payment")
			return nil
		},
	)

	err := coord.Run(context.Background())
	if err == nil {
		t.Fatalf("expected saga to fail due to gateway error")
	}

	// Verify reverse compensation order!
	// Forward: exec_validate -> exec_reserve -> exec_khata_debit -> exec_payment_failed
	// Reverse Compensation: comp_khata_credit_reversal -> comp_reserve_restock -> comp_validate
	expectedLog := []string{
		"exec_validate",
		"exec_reserve",
		"exec_khata_debit",
		"exec_payment_failed",
		"comp_khata_credit_reversal",
		"comp_reserve_restock",
		"comp_validate",
	}

	if len(stepLog) != len(expectedLog) {
		t.Fatalf("execution trace length mismatch:\ngot:  %v\nwant: %v", stepLog, expectedLog)
	}

	for i := range expectedLog {
		if stepLog[i] != expectedLog[i] {
			t.Errorf("trace[%d] = %s, want %s", i, stepLog[i], expectedLog[i])
		}
	}
}

func TestOrderSaga_ContextCancellationCompensates(t *testing.T) {
	var step1Compensated bool
	ctx, cancel := context.WithCancel(context.Background())

	coord := saga.New()
	coord.AddStep(
		"ReserveInventory",
		func(c context.Context) error {
			return nil // Step 1 successfully locks stock
		},
		func(c context.Context) error {
			step1Compensated = true // Must be compensated when step 2 fails
			return nil
		},
	)
	coord.AddStep(
		"ProcessPayment",
		func(c context.Context) error {
			cancel() // Client drops connection during payment
			return c.Err()
		},
		func(c context.Context) error {
			return nil
		},
	)

	err := coord.Run(ctx)
	if err == nil {
		t.Fatalf("expected error due to canceled context")
	}

	if !step1Compensated {
		t.Fatalf("expected Step 1 inventory reservation to be compensated when payment was canceled!")
	}
}

func TestOrderSaga_ObservabilityHook(t *testing.T) {
	var observedEvents []string

	coord := saga.New()
	coord.OnStepEvent(func(name, phase string, err error) {
		observedEvents = append(observedEvents, name+":"+phase)
	})

	coord.AddStep(
		"StepAlpha",
		func(ctx context.Context) error { return nil },
		nil,
	)

	_ = coord.Run(context.Background())

	if len(observedEvents) < 2 {
		t.Fatalf("expected at least 2 events (executing, completed), got %v", observedEvents)
	}
	if observedEvents[0] != "StepAlpha:executing" || observedEvents[1] != "StepAlpha:completed" {
		t.Fatalf("unexpected events order: %v", observedEvents)
	}
}
