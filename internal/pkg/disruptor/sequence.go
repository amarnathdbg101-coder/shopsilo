package disruptor

import "sync/atomic"

// Sequence represents a 64-byte cache-line padded sequence cursor.
// Padding eliminates false sharing between concurrent producer and consumer CPU cores.
type Sequence struct {
	_pad0 [7]uint64
	val   uint64
	_pad1 [7]uint64
}

// NewSequence creates a sequence cursor initialized to the given value.
func NewSequence(initial uint64) *Sequence {
	return &Sequence{val: initial}
}

// Get returns the current sequence value with atomic load semantics.
func (s *Sequence) Get() uint64 {
	return atomic.LoadUint64(&s.val)
}

// Set stores the sequence value with atomic store semantics.
func (s *Sequence) Set(v uint64) {
	atomic.StoreUint64(&s.val, v)
}

// CompareAndSwap executes an atomic compare-and-swap operation.
func (s *Sequence) CompareAndSwap(old, new uint64) bool {
	return atomic.CompareAndSwapUint64(&s.val, old, new)
}

// Add increments the sequence by delta and returns the new value.
func (s *Sequence) Add(delta uint64) uint64 {
	return atomic.AddUint64(&s.val, delta)
}
