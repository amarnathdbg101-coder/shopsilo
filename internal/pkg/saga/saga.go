package saga

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// StepAction represents a forward or compensating workload.
type StepAction func(ctx context.Context) error

// Step defines a named transaction unit and its corresponding rollback action.
type Step struct {
	Name       string
	Execute    StepAction
	Compensate StepAction
}

// LogRecord tracks individual step outcome during saga execution.
type LogRecord struct {
	StepName        string
	Executed        bool
	ExecutionErr    error
	Compensated     bool
	CompensationErr error
}

// Coordinator orchestrates sequential steps with backward compensation upon failure.
type Coordinator struct {
	mu     sync.Mutex
	steps  []Step
	onStep func(name string, phase string, err error)
}

// New creates an empty saga Coordinator.
func New() *Coordinator {
	return &Coordinator{}
}

// AddStep registers a new step with an execution workload and compensating rollback action.
func (c *Coordinator) AddStep(name string, execute, compensate StepAction) *Coordinator {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.steps = append(c.steps, Step{
		Name:       name,
		Execute:    execute,
		Compensate: compensate,
	})
	return c
}

// OnStepEvent registers an observability hook for step execution and compensation events.
func (c *Coordinator) OnStepEvent(fn func(name string, phase string, err error)) *Coordinator {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStep = fn
	return c
}

// Run executes the saga sequentially. If any step fails, compensation actions for all
// previously completed steps are executed in strict reverse order.
func (c *Coordinator) Run(ctx context.Context) error {
	c.mu.Lock()
	steps := make([]Step, len(c.steps))
	copy(steps, c.steps)
	onStep := c.onStep
	c.mu.Unlock()

	if len(steps) == 0 {
		return ErrEmptySaga
	}

	executedIndices := make([]int, 0, len(steps))
	var forwardErr error
	var failedIndex int

	// 1. Forward Execution Loop
	for i, step := range steps {
		if err := ctx.Err(); err != nil {
			forwardErr = err
			failedIndex = i
			break
		}

		if onStep != nil {
			onStep(step.Name, "executing", nil)
		}

		err := safeExecute(ctx, step.Execute)
		if err != nil {
			forwardErr = fmt.Errorf("step '%s' failed: %w", step.Name, err)
			failedIndex = i
			if onStep != nil {
				onStep(step.Name, "failed", err)
			}
			break
		}

		executedIndices = append(executedIndices, i)
		if onStep != nil {
			onStep(step.Name, "completed", nil)
		}
	}

	// If all steps succeeded, we are done
	if forwardErr == nil {
		return nil
	}

	// 2. Backward Compensation Loop (in reverse order)
	var compErrors []string
	// Use detached or non-canceled context for compensations if original context was canceled
	compCtx := context.Background()

	for i := len(executedIndices) - 1; i >= 0; i-- {
		stepIdx := executedIndices[i]
		step := steps[stepIdx]

		if step.Compensate == nil {
			continue
		}

		if onStep != nil {
			onStep(step.Name, "compensating", nil)
		}

		err := safeExecute(compCtx, step.Compensate)
		if err != nil {
			compErrors = append(compErrors, fmt.Sprintf("step '%s' compensation error: %v", step.Name, err))
			if onStep != nil {
				onStep(step.Name, "compensation_failed", err)
			}
		} else {
			if onStep != nil {
				onStep(step.Name, "compensated", nil)
			}
		}
	}

	if len(compErrors) > 0 {
		return fmt.Errorf("%w (forward error: %v; compensations failed:\n  - %s)",
			ErrCompensationFailed, forwardErr, strings.Join(compErrors, "\n  - "))
	}

	_ = failedIndex
	return fmt.Errorf("%w: %v", ErrSagaFailed, forwardErr)
}

func safeExecute(ctx context.Context, action StepAction) (err error) {
	if action == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("action panic: %v", r)
		}
	}()
	return action(ctx)
}
