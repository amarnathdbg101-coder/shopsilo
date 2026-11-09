package worker_test

import (
	"context"
	"testing"
	"time"

	"shopMe/internal/worker"

	"go.uber.org/zap"
)

func TestAnomalyWorker_Lifecycle(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should start and stop cleanly within the context timeout without panicking
	worker.StartAnomalyWorker(ctx, nil, nil, logger)
}
