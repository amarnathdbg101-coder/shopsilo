package services_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"shopMe/internal/handler/services"

	gowshub "github.com/amarnathdbg101-coder/go/go-wshub"
)

type mockWSConn struct {
	mu       sync.Mutex
	messages [][]byte
	closed   bool
	err      error
}

func (m *mockWSConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, append([]byte(nil), data...))
	return nil
}

func (m *mockWSConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return nil
}

func (m *mockWSConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockWSConn) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func TestWebSocketHub_Integration(t *testing.T) {
	wsHub := services.GetWebSocketHub(nil)

	mock1 := &mockWSConn{}
	mock2 := &mockWSConn{}

	sc1 := gowshub.NewSafeConn(mock1)
	sc2 := gowshub.NewSafeConn(mock2)

	// Register shop and storefront
	wsHub.Register("shop_test_101", sc1)
	wsHub.RegisterStorefront("super-mart-slug", sc2)

	// Broadcast to shop
	wsHub.BroadcastToShop("shop_test_101", "NEW_ORDER", map[string]string{"id": "order_123"})
	time.Sleep(50 * time.Millisecond)

	if mock1.count() != 1 {
		t.Fatalf("expected mock1 to receive 1 message, got %d", mock1.count())
	}
	if mock2.count() != 0 {
		t.Fatalf("expected mock2 to receive 0 messages, got %d", mock2.count())
	}

	// Broadcast to storefront
	wsHub.BroadcastToStorefront("super-mart-slug", "CATALOG_UPDATED", map[string]string{"action": "refresh"})
	time.Sleep(50 * time.Millisecond)

	if mock2.count() != 1 {
		t.Fatalf("expected mock2 to receive 1 message, got %d", mock2.count())
	}

	// Unregister
	wsHub.Unregister("shop_test_101", sc1)
	wsHub.UnregisterStorefront("super-mart-slug", sc2)

	// Broadcast again
	wsHub.BroadcastToShop("shop_test_101", "STOCK_CHANGED", nil)
	time.Sleep(50 * time.Millisecond)

	if mock1.count() != 1 {
		t.Fatalf("expected no new messages after unregister, got %d", mock1.count())
	}
}

func TestWebSocketHub_GhostConnectionAutoPruning(t *testing.T) {
	wsHub := services.GetWebSocketHub(nil)

	failingMock := &mockWSConn{err: errors.New("connection reset by peer")}
	sc := gowshub.NewSafeConn(failingMock)

	wsHub.Register("shop_failing", sc)
	wsHub.BroadcastToShop("shop_failing", "PING", nil)
	time.Sleep(50 * time.Millisecond)

	// Broken connection should be closed
	failingMock.mu.Lock()
	closed := failingMock.closed
	failingMock.mu.Unlock()

	if !closed {
		t.Fatal("expected failing connection to be closed automatically on write error")
	}
}
