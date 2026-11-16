package gowshub_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gowshub "shopMe/internal/pkg/wshub"
)

// mockConn simulates a WebSocket connection with thread-safe capture of written frames.
type mockConn struct {
	mu           sync.Mutex
	messages     [][]byte
	controlTypes []int
	closed       bool
	failWrite    bool
}

func (m *mockConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWrite {
		return errors.New("broken pipe / client disconnected")
	}
	m.messages = append(m.messages, append([]byte(nil), data...))
	return nil
}

func (m *mockConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failWrite {
		return errors.New("broken pipe / control failed")
	}
	m.controlTypes = append(m.controlTypes, messageType)
	return nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func TestSafeConn_ConcurrentWrites(t *testing.T) {
	mock := &mockConn{}
	sc := gowshub.NewSafeConn(mock)

	var wg sync.WaitGroup
	const totalWrites = 500

	for i := 0; i < totalWrites; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			_ = sc.WriteMessage(gowshub.TextMessage, []byte("data"))
		}(i)
	}

	wg.Wait()

	if mock.count() != totalWrites {
		t.Fatalf("expected %d messages, got %d", totalWrites, mock.count())
	}
}

func TestHub_JoinAndBroadcast(t *testing.T) {
	hub := gowshub.New()
	defer hub.Close()

	client1 := gowshub.NewSafeConn(&mockConn{})
	client2 := gowshub.NewSafeConn(&mockConn{})
	client3 := gowshub.NewSafeConn(&mockConn{})

	_ = hub.Join("shop:101", client1)
	_ = hub.Join("shop:101", client2)
	_ = hub.Join("shop:202", client3)

	if hub.RoomSize("shop:101") != 2 {
		t.Fatalf("expected 2 clients in shop:101, got %d", hub.RoomSize("shop:101"))
	}
	if hub.RoomSize("shop:202") != 1 {
		t.Fatalf("expected 1 client in shop:202, got %d", hub.RoomSize("shop:202"))
	}
	if hub.TotalConnections() != 3 {
		t.Fatalf("expected 3 total connections, got %d", hub.TotalConnections())
	}

	// Broadcast event to shop:101 only
	err := hub.BroadcastEvent("shop:101", "ORDER_PLACED", map[string]string{"id": "ord_1"})
	if err != nil {
		t.Fatalf("failed to broadcast event: %v", err)
	}

	// Wait briefly for async goroutines
	time.Sleep(50 * time.Millisecond)

	m1 := client1.Raw().(*mockConn)
	m2 := client2.Raw().(*mockConn)
	m3 := client3.Raw().(*mockConn)

	if m1.count() != 1 {
		t.Fatalf("expected client1 to receive 1 message, got %d", m1.count())
	}
	if m2.count() != 1 {
		t.Fatalf("expected client2 to receive 1 message, got %d", m2.count())
	}
	if m3.count() != 0 {
		t.Fatalf("expected client3 to receive 0 messages, got %d", m3.count())
	}
}

func TestHub_GhostPruningOnWriteFailure(t *testing.T) {
	hub := gowshub.New()
	defer hub.Close()

	failingMock := &mockConn{failWrite: true}
	client := gowshub.NewSafeConn(failingMock)

	_ = hub.Join("room:alerts", client)
	if hub.RoomSize("room:alerts") != 1 {
		t.Fatalf("expected 1 client in room, got %d", hub.RoomSize("room:alerts"))
	}

	// Broadcast causes write error
	hub.Broadcast("room:alerts", gowshub.TextMessage, []byte("urgent alert"))

	// Give background worker time to handle write error and unregister
	time.Sleep(50 * time.Millisecond)

	if hub.RoomSize("room:alerts") != 0 {
		t.Fatalf("expected broken client to be pruned from room, got %d", hub.RoomSize("room:alerts"))
	}
	if !failingMock.closed {
		t.Fatal("expected failing connection to be closed")
	}
}

func TestHub_LeaveAndLeaveAll(t *testing.T) {
	hub := gowshub.New()
	defer hub.Close()

	client := gowshub.NewSafeConn(&mockConn{})
	_ = hub.Join("room:a", client)
	_ = hub.Join("room:b", client)

	if hub.TotalConnections() != 1 {
		t.Fatalf("expected 1 connection, got %d", hub.TotalConnections())
	}

	hub.Leave("room:a", client)
	if hub.RoomSize("room:a") != 0 {
		t.Fatalf("expected room:a to be empty, got %d", hub.RoomSize("room:a"))
	}
	if hub.RoomSize("room:b") != 1 {
		t.Fatalf("expected room:b to have 1 client, got %d", hub.RoomSize("room:b"))
	}

	hub.LeaveAll(client)
	if hub.RoomSize("room:b") != 0 {
		t.Fatalf("expected room:b to be empty after LeaveAll, got %d", hub.RoomSize("room:b"))
	}
	if hub.TotalConnections() != 0 {
		t.Fatalf("expected 0 total connections after LeaveAll, got %d", hub.TotalConnections())
	}
}

func TestHeartbeat_PingLoop(t *testing.T) {
	mock := &mockConn{}
	sc := gowshub.NewSafeConn(mock)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pingCount atomic.Int32
	stop := gowshub.StartHeartbeat(ctx, sc, 10*time.Millisecond, 50*time.Millisecond)
	defer stop()

	time.Sleep(35 * time.Millisecond)
	stop()

	mock.mu.Lock()
	count := len(mock.controlTypes)
	mock.mu.Unlock()

	if count < 2 {
		t.Fatalf("expected at least 2 ping frames, got %d", count)
	}
	_ = pingCount
}
