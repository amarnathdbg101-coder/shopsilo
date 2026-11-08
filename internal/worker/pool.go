package worker

import (
	"context"

	goworkerpool "github.com/amarnathdbg101-coder/go/go-workerpool"
	"go.uber.org/zap"
)

var defaultTaskPool = goworkerpool.New(8, 512, goworkerpool.WithOnPanic(func(recovered any) {
	zap.L().Error("async background task panic recovered", zap.Any("panic", recovered))
}))

// SubmitTask enqueues an asynchronous background job with bounded concurrency and panic isolation.
// If the worker pool is closed or queue is exhausted, it returns an error.
func SubmitTask(job func()) error {
	return defaultTaskPool.Submit(func(ctx context.Context) {
		job()
	})
}

// DrainTaskPool drains all in-flight jobs and shuts down the background task pool during graceful shutdown.
func DrainTaskPool() {
	defaultTaskPool.Stop()
}
