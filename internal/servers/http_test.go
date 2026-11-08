package servers_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"shopMe/internal/servers"

	"go.uber.org/zap"
)

func TestNewServer(t *testing.T) {
	mux := http.NewServeMux()
	srv := servers.NewServer("8080", mux)

	if srv.Addr != "0.0.0.0:8080" {
		t.Errorf("expected Addr 0.0.0.0:8080, got %s", srv.Addr)
	}
	if srv.ReadTimeout != 30*time.Second {
		t.Errorf("expected ReadTimeout 30s, got %v", srv.ReadTimeout)
	}
}

func TestBuildShutdownCoordinator_PhasedExecution(t *testing.T) {
	mux := http.NewServeMux()
	srv := servers.NewServer("0", mux)
	logger := zap.NewNop()

	var workerCalled bool
	var wg sync.WaitGroup
	wg.Add(1)

	stopApp := func() {
		workerCalled = true
		wg.Done()
	}

	coordinator := servers.BuildShutdownCoordinator(srv, stopApp, logger, &wg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Programmatically trigger phases sequentially (Phase 1 -> Phase 2)
	err := coordinator.Trigger(ctx)
	if err != nil {
		t.Fatalf("unexpected error during coordinator trigger: %v", err)
	}

	if !workerCalled {
		t.Errorf("expected worker stopApp to be invoked during phased shutdown")
	}
}
