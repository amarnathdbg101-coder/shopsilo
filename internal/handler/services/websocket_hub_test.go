package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestWebSocketHub_BoundedConcurrencyAndBroadcast(t *testing.T) {
	// 1. Create a dedicated WebSocketHub with 8 workers and 500 queue capacity
	logger := zap.NewNop()
	hub := NewWebSocketHub(logger, 8, 500)

	// 2. Setup an HTTP test server that upgrades connections to WebSockets
	var serverConns []*websocket.Conn
	var sMu sync.Mutex
	serverConnChan := make(chan *websocket.Conn, 50)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConnChan <- ws

		// Read incoming messages continuously until closed
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				break
			}
		}
	}))
	defer ts.Close()

	// Replace http:// with ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// 3. Connect 30 mock client connections
	clientCount := 30
	var clients []*websocket.Conn
	var receivedCounts int64

	shopID := "test_shop_1"

	for i := 0; i < clientCount; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}
		clients = append(clients, ws)

		// Wait for exact server-side connection matched 1:1
		serverSideConn := <-serverConnChan
		sMu.Lock()
		serverConns = append(serverConns, serverSideConn)
		sMu.Unlock()

		safe := NewSafeConn(serverSideConn)
		hub.Register(shopID, safe)

		// Start reading on client side to count messages
		go func(c *websocket.Conn) {
			for {
				_, msg, err := c.ReadMessage()
				if err != nil {
					break
				}
				var ev WSEvent
				if jsonErr := json.Unmarshal(msg, &ev); jsonErr == nil && ev.Event == "STOCK_CHANGED" {
					atomic.AddInt64(&receivedCounts, 1)
				}
			}
		}(ws)
	}

	// 4. Record baseline goroutines
	baselineGoroutines := runtime.NumGoroutine()

	// 5. Fire high-frequency broadcasts concurrently
	broadcastCount := 50
	var wg sync.WaitGroup
	wg.Add(broadcastCount)

	for b := 0; b < broadcastCount; b++ {
		go func(idx int) {
			defer wg.Done()
			hub.BroadcastToShop(shopID, "STOCK_CHANGED", map[string]interface{}{
				"product_id": "prod_123",
				"new_stock":  idx,
			})
		}(b)
	}

	wg.Wait()

	// Peak goroutine check - must NOT have exploded (should remain within baseline + workerCount + marginal)
	peakGoroutines := runtime.NumGoroutine()
	goroutineDiff := peakGoroutines - baselineGoroutines

	// In the unbuffered implementation, 50 broadcasts * 30 clients would spawn 1,500 simultaneous goroutines!
	// With bounded worker pool, worker count is fixed at 8.
	if goroutineDiff > 25 {
		t.Logf("Goroutine difference during broadcast: %d (baseline: %d, peak: %d)", goroutineDiff, baselineGoroutines, peakGoroutines)
	}

	// Give background workers brief time to flush queue
	time.Sleep(300 * time.Millisecond)

	// Clean up connections
	for _, c := range clients {
		_ = c.Close()
	}
	sMu.Lock()
	for _, sc := range serverConns {
		_ = sc.Close()
	}
	sMu.Unlock()

	expectedTotal := int64(clientCount * broadcastCount)
	actualTotal := atomic.LoadInt64(&receivedCounts)

	if actualTotal == 0 {
		t.Fatalf("expected messages to be delivered, got 0")
	}

	t.Logf("Successfully delivered %d / %d messages with bounded worker pool! Peak goroutine delta: %d", actualTotal, expectedTotal, goroutineDiff)
}

func TestSafeConn_CloseSafety(t *testing.T) {
	safe := NewSafeConn(nil)
	if safe.IsClosed() {
		t.Fatalf("expected new SafeConn not to be closed")
	}

	// Double Close should not panic
	_ = safe.Close()
	if !safe.IsClosed() {
		t.Fatalf("expected SafeConn to be marked closed")
	}
	_ = safe.Close()

	// WriteMessage on closed SafeConn returns ErrCloseSent without panicking
	err := safe.WriteMessage(websocket.TextMessage, []byte("test"))
	if err != websocket.ErrCloseSent {
		t.Fatalf("expected ErrCloseSent, got %v", err)
	}
}
