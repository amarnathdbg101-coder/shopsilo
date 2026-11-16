package deadlock_test

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"shopMe/internal/pkg/deadlock"
)

func TestNormalLockUnlock(t *testing.T) {
	var mu deadlock.Mutex
	mu.Lock()
	val := 42
	val++
	mu.Unlock()

	if val != 43 {
		t.Fatalf("expected val 43, got %d", val)
	}
}

func TestTryLock(t *testing.T) {
	var mu deadlock.Mutex
	if !mu.TryLock() {
		t.Fatal("expected TryLock to succeed")
	}
	if mu.TryLock() {
		t.Fatal("expected second TryLock to fail")
	}
	mu.Unlock()

	if !mu.TryLock() {
		t.Fatal("expected TryLock to succeed after Unlock")
	}
	mu.Unlock()
}

func TestRWMutex(t *testing.T) {
	var rw deadlock.RWMutex
	rw.RLock()
	rw.RLock()
	rw.RUnlock()
	rw.RUnlock()

	rw.Lock()
	rw.Unlock()
}

func TestPotentialDeadlockDetection(t *testing.T) {
	var deadlockAlerts int32
	var alertMsg string
	var msgMu sync.Mutex

	deadlock.SetConfig(deadlock.Config{
		Timeout: 500 * time.Millisecond,
		OnPotentialDeadlock: func(msg string) {
			atomic.AddInt32(&deadlockAlerts, 1)
			msgMu.Lock()
			alertMsg = msg
			msgMu.Unlock()
		},
	})

	var muA deadlock.Mutex
	var muB deadlock.Mutex

	// Path 1: acquire A, then acquire B
	muA.Lock()
	muB.Lock()
	muB.Unlock()
	muA.Unlock()

	// Path 2: Invert order! Acquire B, then attempt A
	// This creates an AB-BA inversion in the directed graph!
	muB.Lock()
	muA.Lock() // Cycle detected here!
	muA.Unlock()
	muB.Unlock()

	if atomic.LoadInt32(&deadlockAlerts) == 0 {
		t.Fatal("expected potential deadlock alert for AB-BA lock inversion")
	}

	msgMu.Lock()
	defer msgMu.Unlock()
	if !strings.Contains(alertMsg, "AB-BA Deadlock") {
		t.Errorf("expected alert message to contain 'AB-BA Deadlock', got: %s", alertMsg)
	}
}

func TestLockTimeoutWarning(t *testing.T) {
	var timeoutCount int32
	deadlock.SetConfig(deadlock.Config{
		Timeout: 20 * time.Millisecond,
		OnTimeout: func(msg string) {
			atomic.AddInt32(&timeoutCount, 1)
		},
	})

	var mu deadlock.Mutex
	mu.Lock()

	// Launch another goroutine that tries to lock mu and waits beyond 20ms
	done := make(chan struct{})
	go func() {
		mu.Lock()
		mu.Unlock()
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	mu.Unlock()
	<-done

	if atomic.LoadInt32(&timeoutCount) == 0 {
		t.Fatal("expected timeout alert when lock is held longer than threshold")
	}
}

