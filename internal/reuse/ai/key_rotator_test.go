package ai

import (
	"testing"
	"time"
)

func TestKeyRotator_RoundRobin(t *testing.T) {
	keys := []string{"key-1", "key-2", "key-3"}
	rotator := NewKeyRotator(keys)

	if !rotator.HasKeys() {
		t.Fatal("expected HasKeys to be true")
	}

	k1, err := rotator.GetNextKey()
	if err != nil || k1 != "key-1" {
		t.Fatalf("expected key-1, got %s, err: %v", k1, err)
	}

	k2, err := rotator.GetNextKey()
	if err != nil || k2 != "key-2" {
		t.Fatalf("expected key-2, got %s, err: %v", k2, err)
	}

	k3, err := rotator.GetNextKey()
	if err != nil || k3 != "key-3" {
		t.Fatalf("expected key-3, got %s, err: %v", k3, err)
	}

	k4, err := rotator.GetNextKey()
	if err != nil || k4 != "key-1" {
		t.Fatalf("expected key-1 on wrap-around, got %s, err: %v", k4, err)
	}
}

func TestKeyRotator_RateLimitCooldown(t *testing.T) {
	keys := []string{"key-1", "key-2"}
	rotator := NewKeyRotator(keys)

	// Mark key-1 as rate limited for 100ms
	rotator.MarkRateLimited("key-1", 100*time.Millisecond)

	// Next call should skip key-1 and return key-2
	k, err := rotator.GetNextKey()
	if err != nil || k != "key-2" {
		t.Fatalf("expected key-2, got %s, err: %v", k, err)
	}

	// Wait for cooldown to expire
	time.Sleep(120 * time.Millisecond)

	// Next call should pick key-1 again as it recovered
	kAfter, err := rotator.GetNextKey()
	if err != nil || kAfter != "key-1" {
		t.Fatalf("expected key-1 after cooldown, got %s, err: %v", kAfter, err)
	}
}
