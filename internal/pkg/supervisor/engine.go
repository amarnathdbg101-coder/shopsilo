package supervisor

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

type workerState struct {
	spec         WorkerSpec
	cancel       context.CancelFunc
	restarts     []time.Time
	status       string
	lastError    error
	runningState bool
}

// Supervisor orchestrates and monitors goroutine lifecycles with Erlang/OTP recovery semantics.
type Supervisor struct {
	cfg     Config
	mu      sync.Mutex
	workers map[string]*workerState

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	running bool
	closed  bool
}

// New creates a new Supervisor instance with the supplied options.
func New(opts ...Option) *Supervisor {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Supervisor{
		cfg:     cfg,
		workers: make(map[string]*workerState),
	}
}

// Add registers a worker specification with the supervisor.
func (s *Supervisor) Add(spec WorkerSpec) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workers[spec.Name]; exists {
		return ErrWorkerAlreadyRegistered
	}
	if spec.Policy == "" {
		spec.Policy = PolicyPermanent
	}

	state := &workerState{
		spec:     spec,
		restarts: make([]time.Time, 0),
		status:   "STOPPED",
	}
	s.workers[spec.Name] = state

	// If supervisor already running, immediately start the newly added worker
	if s.running && !s.closed {
		s.startWorkerLocked(state)
	}

	return nil
}

// Supervise is a convenience helper to add a worker with default permanent policy.
func (s *Supervisor) Supervise(name string, fn WorkerFunc) error {
	return s.Add(WorkerSpec{
		Name:   name,
		Func:   fn,
		Policy: PolicyPermanent,
	})
}

// Start launches all supervised workers and blocks until ctx is cancelled or a fatal crash loop occurs.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	if s.closed {
		s.mu.Unlock()
		return ErrSupervisorClosed
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true

	for _, w := range s.workers {
		s.startWorkerLocked(w)
	}
	s.mu.Unlock()

	<-s.ctx.Done()

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	s.wg.Wait()
	return nil
}

// Stop gracefully stops all workers and the supervisor.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	if !s.running || s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	s.wg.Wait()
	return nil
}

func (s *Supervisor) startWorkerLocked(w *workerState) {
	if w.runningState {
		return
	}

	workerCtx, cancel := context.WithCancel(s.ctx)
	w.cancel = cancel
	w.runningState = true
	w.status = "RUNNING"

	s.wg.Add(1)
	go s.runWorker(w, workerCtx)
}

func (s *Supervisor) runWorker(w *workerState, ctx context.Context) {
	defer s.wg.Done()

	var err error
	var panicked bool

	// Panic isolation enclosure
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				stack := debug.Stack()
				err = fmt.Errorf("panic: %v", r)
				if s.cfg.OnWorkerPanic != nil {
					s.cfg.OnWorkerPanic(w.spec.Name, r, stack)
				}
			}
		}()
		err = w.spec.Func(ctx)
	}()

	s.mu.Lock()
	w.runningState = false
	w.lastError = err

	// If supervisor is shutting down, exit cleanly without restart
	if s.closed || (s.ctx != nil && s.ctx.Err() != nil) {
		w.status = "STOPPED"
		s.mu.Unlock()
		return
	}

	if s.cfg.OnWorkerTerminated != nil {
		s.cfg.OnWorkerTerminated(w.spec.Name, err)
	}

	// Evaluate restart policy
	shouldRestart := false
	switch w.spec.Policy {
	case PolicyPermanent:
		shouldRestart = true
	case PolicyTransient:
		shouldRestart = (err != nil || panicked)
	case PolicyTemporary:
		shouldRestart = false
	}

	if !shouldRestart {
		w.status = "STOPPED"
		s.mu.Unlock()
		return
	}

	// Check crash loop protection
	now := time.Now()
	cutoff := now.Add(-s.cfg.RestartWindow)
	recent := make([]time.Time, 0, len(w.restarts)+1)
	for _, t := range w.restarts {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	w.restarts = recent

	if len(recent) > s.cfg.MaxRestarts {
		w.status = "CRASHED"
		s.mu.Unlock()
		// Abort supervisor on crash loop
		if s.cancel != nil {
			s.cancel()
		}
		return
	}

	s.mu.Unlock()

	// Apply backoff if configured
	if w.spec.Backoff > 0 {
		select {
		case <-time.After(w.spec.Backoff):
		case <-ctx.Done():
			return
		}
	}

	// Handle Strategy: OneForOne vs OneForAll
	s.mu.Lock()
	if s.cfg.Strategy == StrategyOneForAll {
		// Stop all other workers first
		for _, other := range s.workers {
			if other != w && other.cancel != nil {
				other.cancel()
			}
		}
	}

	// Restart
	if s.running && !s.closed {
		s.startWorkerLocked(w)
	}
	s.mu.Unlock()
}
