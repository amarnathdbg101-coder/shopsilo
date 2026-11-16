package golru

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestLRU_BasicAndEviction(t *testing.T) {
	var evictedKeys []string
	c, err := New(2, func(key string, val int) {
		evictedKeys = append(evictedKeys, key)
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	c.Set("k1", 1, 0)
	c.Set("k2", 2, 0)

	// Access k1 so k1 becomes MRU and k2 becomes LRU
	val, ok := c.Get("k1")
	if !ok || val != 1 {
		t.Fatalf("expected k1=1, got %v", val)
	}

	// Add k3 -> Capacity 2 exceeded -> k2 must be evicted!
	c.Set("k3", 3, 0)

	if c.Contains("k2") {
		t.Errorf("expected k2 to be evicted")
	}
	if !c.Contains("k1") || !c.Contains("k3") {
		t.Errorf("expected k1 and k3 to remain")
	}

	if len(evictedKeys) != 1 || evictedKeys[0] != "k2" {
		t.Errorf("expected eviction callback for 'k2', got %v", evictedKeys)
	}
}

func TestLRU_TTLExpiration(t *testing.T) {
	var evictedKey string
	c, _ := New(5, func(key string, val string) {
		evictedKey = key
	})

	c.Set("token", "secret123", 40*time.Millisecond)

	// Immediate access
	val, ok := c.Get("token")
	if !ok || val != "secret123" {
		t.Fatalf("expected secret123, got %v", val)
	}

	time.Sleep(70 * time.Millisecond)

	// Expired access
	_, ok = c.Get("token")
	if ok {
		t.Errorf("expected token to be expired")
	}

	if evictedKey != "token" {
		t.Errorf("expected eviction callback for 'token', got %s", evictedKey)
	}
}

func TestLRU_PeekAndKeysOrder(t *testing.T) {
	c, _ := New[string, int](3, nil)

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	// Keys should be c, b, a (MRU to LRU)
	keys := c.Keys()
	expected := []string{"c", "b", "a"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("expected keys %v, got %v", expected, keys)
	}

	// Peek "a" should NOT move it to front
	val, ok := c.Peek("a")
	if !ok || val != 1 {
		t.Fatalf("expected 1 from peek, got %v", val)
	}

	keysAfterPeek := c.Keys()
	if !reflect.DeepEqual(keysAfterPeek, expected) {
		t.Errorf("peek modified LRU order! expected %v, got %v", expected, keysAfterPeek)
	}
}

func TestLRU_ConcurrentStress(t *testing.T) {
	c, _ := New[int, int](50, nil)
	var wg sync.WaitGroup
	workers := 10
	iterations := 200

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := j % 100
				c.Set(key, key*10, 0)
				_, _ = c.Get(key)
			}
		}(i)
	}

	wg.Wait()

	if c.Len() > c.Cap() {
		t.Errorf("cache size %d exceeded capacity %d", c.Len(), c.Cap())
	}
}
