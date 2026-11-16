package slidingwindow

import (
	"sync"
	"time"
)

type bucket struct {
	timestamp int64 // bucket index (time / bucketDuration)
	count     int64
}

// Window tracks occurrences over a rolling time window using a ring buffer of sub-buckets.
type Window struct {
	mu             sync.RWMutex
	windowSize     time.Duration
	bucketDuration time.Duration
	bucketCount    int
	buckets        []bucket
}

// New creates a new sliding window counter with specified window size and number of sub-buckets.
// Example: New(60*time.Second, 60) creates a 1-minute window with 1-second resolution.
func New(windowSize time.Duration, bucketCount int) (*Window, error) {
	if windowSize <= 0 {
		return nil, ErrInvalidWindow
	}
	if bucketCount <= 0 {
		return nil, ErrInvalidBuckets
	}

	bucketDuration := windowSize / time.Duration(bucketCount)
	if bucketDuration <= 0 {
		bucketDuration = time.Millisecond
	}

	return &Window{
		windowSize:     windowSize,
		bucketDuration: bucketDuration,
		bucketCount:    bucketCount,
		buckets:        make([]bucket, bucketCount),
	}, nil
}

func (w *Window) getBucketIndex(now time.Time) (int64, int) {
	currentBucket := now.UnixNano() / int64(w.bucketDuration)
	idx := int(currentBucket % int64(w.bucketCount))
	if idx < 0 {
		idx = -idx
	}
	return currentBucket, idx
}

// Inc increments the current active bucket by delta.
func (w *Window) Inc(delta int64) {
	if delta <= 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	currentBucket, idx := w.getBucketIndex(now)

	b := &w.buckets[idx]
	if b.timestamp != currentBucket {
		// New cycle, reset bucket
		b.timestamp = currentBucket
		b.count = delta
	} else {
		b.count += delta
	}
}

// Count returns the sum of counts across all valid non-expired buckets in the sliding window.
func (w *Window) Count() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	now := time.Now()
	currentBucket := now.UnixNano() / int64(w.bucketDuration)
	oldestValidBucket := currentBucket - int64(w.bucketCount) + 1

	var total int64
	for i := 0; i < w.bucketCount; i++ {
		b := &w.buckets[i]
		if b.timestamp >= oldestValidBucket && b.timestamp <= currentBucket {
			total += b.count
		}
	}
	return total
}

// Rate computes the average rate of events per second across the active window.
func (w *Window) Rate() float64 {
	count := w.Count()
	seconds := w.windowSize.Seconds()
	if seconds <= 0 {
		return 0
	}
	return float64(count) / seconds
}

// Allow checks if adding 1 to the window would stay within limit.
// If within limit, increments the counter by 1 and returns true; otherwise returns false.
func (w *Window) Allow(limit int64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	currentBucket, idx := w.getBucketIndex(now)
	oldestValidBucket := currentBucket - int64(w.bucketCount) + 1

	var total int64
	for i := 0; i < w.bucketCount; i++ {
		b := &w.buckets[i]
		if b.timestamp >= oldestValidBucket && b.timestamp <= currentBucket {
			total += b.count
		}
	}

	if total+1 > limit {
		return false
	}

	// Increment
	b := &w.buckets[idx]
	if b.timestamp != currentBucket {
		b.timestamp = currentBucket
		b.count = 1
	} else {
		b.count++
	}

	return true
}

// Reset clears all counters in the sliding window.
func (w *Window) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i := 0; i < w.bucketCount; i++ {
		w.buckets[i] = bucket{}
	}
}

// KeyedLimiter provides multi-tenant rate limiting across dynamic keys (e.g., user IDs, IP addresses).
type KeyedLimiter struct {
	mu             sync.RWMutex
	windows        map[string]*Window
	windowSize     time.Duration
	bucketCount    int
	limit          int64
}

// NewKeyedLimiter creates a multi-tenant sliding window rate limiter.
func NewKeyedLimiter(windowSize time.Duration, bucketCount int, limit int64) (*KeyedLimiter, error) {
	if windowSize <= 0 {
		return nil, ErrInvalidWindow
	}
	if bucketCount <= 0 {
		return nil, ErrInvalidBuckets
	}

	return &KeyedLimiter{
		windows:     make(map[string]*Window),
		windowSize:  windowSize,
		bucketCount: bucketCount,
		limit:       limit,
	}, nil
}

func (kl *KeyedLimiter) getOrCreate(key string) *Window {
	kl.mu.RLock()
	w, exists := kl.windows[key]
	kl.mu.RUnlock()
	if exists {
		return w
	}

	kl.mu.Lock()
	defer kl.mu.Unlock()

	if w, exists := kl.windows[key]; exists {
		return w
	}

	w, _ = New(kl.windowSize, kl.bucketCount)
	kl.windows[key] = w
	return w
}

// Allow determines if an operation for the given key is allowed under the rate limit.
func (kl *KeyedLimiter) Allow(key string) bool {
	w := kl.getOrCreate(key)
	return w.Allow(kl.limit)
}

// Count returns the current count for key.
func (kl *KeyedLimiter) Count(key string) int64 {
	kl.mu.RLock()
	w, exists := kl.windows[key]
	kl.mu.RUnlock()
	if !exists {
		return 0
	}
	return w.Count()
}
