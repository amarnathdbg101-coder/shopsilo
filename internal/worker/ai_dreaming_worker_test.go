package worker_test

import (
	"context"
	"testing"
	"time"

	"shopMe/internal/handler/services"
	"shopMe/internal/worker"

	"go.uber.org/zap"
)

func TestDreamingWorker_ContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	partnerService := services.NewAIPartnerService(nil, nil, logger)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		worker.StartDreamingWorker(ctx, partnerService, logger)
		close(done)
	}()

	// Allow worker to spin up and cancel immediately
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Succeeded in stopping cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("dreaming worker did not stop cleanly on context cancellation")
	}
}
