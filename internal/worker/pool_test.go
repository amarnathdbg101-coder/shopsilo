package worker_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"shopMe/internal/worker"
)

func TestWorkerPoolExecution(t *testing.T) {
	var counter int64
	var wg sync.WaitGroup

	const tasksCount = 50
	wg.Add(tasksCount)

	for i := 0; i < tasksCount; i++ {
		err := worker.SubmitTask(func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
			time.Sleep(2 * time.Millisecond)
		})
		if err != nil {
			t.Fatalf("failed to submit task %d to workerpool: %v", i, err)
		}
	}

	wg.Wait()

	if atomic.LoadInt64(&counter) != tasksCount {
		t.Fatalf("expected counter %d, got %d", tasksCount, counter)
	}

	t.Run("Panic in background task is safely intercepted without killing pool", func(t *testing.T) {
		var postPanicExecuted int32
		var panicWG sync.WaitGroup
		panicWG.Add(2)

		// 1. Task that panics
		worker.SubmitTask(func() {
			defer panicWG.Done()
			panic("simulated fatal worker error")
		})

		// 2. Normal task submitted after panic
		worker.SubmitTask(func() {
			defer panicWG.Done()
			atomic.StoreInt32(&postPanicExecuted, 1)
		})

		panicWG.Wait()

		if atomic.LoadInt32(&postPanicExecuted) != 1 {
			t.Fatalf("expected post-panic task to execute successfully")
		}
	})
}
