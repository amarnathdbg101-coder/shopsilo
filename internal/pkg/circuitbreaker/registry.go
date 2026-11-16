package gocb

import "sync"

// Registry holds a collection of named CircuitBreaker instances.
type Registry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewRegistry initializes an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Get retrieves a circuit breaker by name.
func (r *Registry) Get(name string) (*CircuitBreaker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cb, ok := r.breakers[name]
	return cb, ok
}

// GetOrRegister retrieves an existing circuit breaker or registers a new one.
func (r *Registry) GetOrRegister(name string, cfg Config) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cb, ok := r.breakers[name]; ok {
		return cb
	}

	cfg.Name = name
	cb := New(cfg)
	r.breakers[name] = cb
	return cb
}

// All returns a snapshot of all registered circuit breakers.
func (r *Registry) All() map[string]*CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make(map[string]*CircuitBreaker, len(r.breakers))
	for k, v := range r.breakers {
		res[k] = v
	}
	return res
}
