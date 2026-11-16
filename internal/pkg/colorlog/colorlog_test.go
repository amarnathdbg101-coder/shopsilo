package colorlog

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestColorLog_LevelsAndFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithOutput(&buf),
		WithLevel(LevelInfo),
		WithNoColor(true),
	)

	// Debug should be filtered out
	logger.Debug("debug message", "key", 1)
	if buf.Len() != 0 {
		t.Fatalf("expected debug to be suppressed at INFO level")
	}

	// Info should be rendered
	logger.Info("user logged in", "user_id", 42, "role", "admin")
	output := buf.String()

	if !strings.Contains(output, "[INFO]") {
		t.Fatalf("missing [INFO] badge in: %s", output)
	}
	if !strings.Contains(output, "user logged in") {
		t.Fatalf("missing message in: %s", output)
	}
	if !strings.Contains(output, "user_id=42") || !strings.Contains(output, "role=admin") {
		t.Fatalf("missing key-values in: %s", output)
	}
}

func TestColorLog_WithContext(t *testing.T) {
	var buf bytes.Buffer
	base := New(
		WithOutput(&buf),
		WithNoColor(true),
	)

	reqLogger := base.With("req_id", "REQ-999")
	reqLogger.Warn("rate limit exceeded", "attempt", 5)

	out := buf.String()
	if !strings.Contains(out, "req_id=REQ-999") || !strings.Contains(out, "attempt=5") {
		t.Fatalf("child logger missing context in: %s", out)
	}
}

func TestColorLog_Colors(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithOutput(&buf),
		WithNoColor(false), // colors enabled
	)

	logger.Error("database failure", "error", "connection reset")
	out := buf.String()

	if !strings.Contains(out, redCode) {
		t.Fatalf("expected ANSI red code for ERROR level, got: %s", out)
	}
	if !strings.Contains(out, cyanCode) {
		t.Fatalf("expected ANSI cyan code for keys, got: %s", out)
	}
}

func TestColorLog_Concurrent(t *testing.T) {
	var buf bytes.Buffer
	logger := New(WithOutput(&buf), WithNoColor(true))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.Info("event", "worker_id", id)
		}(i)
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 50 {
		t.Fatalf("expected 50 logged lines, got %d", len(lines))
	}
}
