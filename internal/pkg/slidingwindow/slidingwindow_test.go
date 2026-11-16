package slidingwindow

import (
	"sync"
	"testing"
	"time"
)

func TestSlidingWindow_Basic(t *testing.T) {
	// 500ms window with 5 buckets (100ms each)
	w, err := New(500*time.Millisecond, 5)
	if err != nil {
		t.Fatalf("failed to create window: %v", err)
	}

	w.Inc(5)
	w.Inc(10)

	if w.Count() != 15 {
		t.Fatalf("expected count 15, got %d", w.Count())
	}

	rate := w.Rate()
	// 15 / 0.5s = 30 events/sec
	if rate < 29.0 || rate > 31.0 {
		t.Fatalf("expected rate ~30, got %f", rate)
	}
}

func TestSlidingWindow_Expiration(t *testing.T) {
	// 100ms window with 5 buckets (20ms each)
	w, _ := New(100*time.Millisecond, 5)

	w.Inc(10)
	if w.Count() != 10 {
		t.Fatalf("expected count 10 before sleep")
	}

	// Sleep past the window duration
	time.Sleep(150 * time.Millisecond)

	if w.Count() != 0 {
		t.Fatalf("expected count 0 after window expiration, got %d", w.Count())
	}
}

func TestSlidingWindow_Allow(t *testing.T) {
	w, _ := New(200*time.Millisecond, 5)

	// Limit = 3
	if !w.Allow(3) {
		t.Fatalf("expected allow 1")
	}
	if !w.Allow(3) {
		t.Fatalf("expected allow 2")
	}
	if !w.Allow(3) {
		t.Fatalf("expected allow 3")
	}
	// 4th request must be rejected
	if w.Allow(3) {
		t.Fatalf("expected request 4 to be rejected")
	}

	if w.Count() != 3 {
		t.Fatalf("expected count 3, got %d", w.Count())
	}
}

func TestKeyedLimiter(t *testing.T) {
	// Limit = 2 per key
	kl, err := NewKeyedLimiter(200*time.Millisecond, 5, 2)
	if err != nil {
		t.Fatalf("failed to create keyed limiter: %v", err)
	}

	// User A
	if !kl.Allow("user-a") || !kl.Allow("user-a") {
		t.Fatalf("expected user-a to be allowed 2 times")
	}
	if kl.Allow("user-a") {
		t.Fatalf("expected user-a 3rd request to be denied")
	}

	// User B is independent and should be allowed
	if !kl.Allow("user-b") {
		t.Fatalf("expected user-b to be allowed independently")
	}
}

func TestSlidingWindow_Concurrent(t *testing.T) {
	w, _ := New(1*time.Second, 10)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Inc(1)
			w.Count()
			w.Rate()
		}()
	}
	wg.Wait()

	if w.Count() != 50 {
		t.Fatalf("expected count 50, got %d", w.Count())
	}
}
