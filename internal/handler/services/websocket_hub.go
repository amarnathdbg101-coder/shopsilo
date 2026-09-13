package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WSEvent struct {
	Event     string      `json:"event"` // 'NEW_RESERVATION', 'NEW_POS_SALE', 'KHATA_PAYMENT'
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
}

type WebSocketHub struct {
	mu          sync.RWMutex
	connections map[string][]*websocket.Conn // map[shopID][]*Conn
	logger      *zap.Logger
}

var (
	globalWSHub *WebSocketHub
	wsHubOnce   sync.Once
)

func GetWebSocketHub(logger *zap.Logger) *WebSocketHub {
	wsHubOnce.Do(func() {
		globalWSHub = &WebSocketHub{
			connections: make(map[string][]*websocket.Conn),
			logger:      logger,
		}
	})
	return globalWSHub
}

func (h *WebSocketHub) Register(shopID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connections[shopID] = append(h.connections[shopID], conn)
	if h.logger != nil {
		h.logger.Info("websocket client connected to shop channel", zap.String("shop_id", shopID), zap.Int("total_connections", len(h.connections[shopID])))
	}
}

func (h *WebSocketHub) Unregister(shopID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns := h.connections[shopID]
	for i, c := range conns {
		if c == conn {
			_ = conn.Close()
			h.connections[shopID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}

	if len(h.connections[shopID]) == 0 {
		delete(h.connections, shopID)
	}
}

// BroadcastToShop sends real-time JSON alert to all active counter devices of that shop in 0ms!
func (h *WebSocketHub) BroadcastToShop(shopID string, eventType string, payload interface{}) {
	h.mu.RLock()
	conns, exists := h.connections[shopID]
	if !exists || len(conns) == 0 {
		h.mu.RUnlock()
		return
	}
	// Copy slices to avoid holding lock during network I/O
	activeConns := make([]*websocket.Conn, len(conns))
	copy(activeConns, conns)
	h.mu.RUnlock()

	event := WSEvent{
		Event:     eventType,
		Payload:   payload,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to marshal websocket event", zap.Error(err))
		}
		return
	}

	for _, conn := range activeConns {
		go func(c *websocket.Conn) {
			_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.WriteMessage(websocket.TextMessage, eventBytes); err != nil {
				if h.logger != nil {
					h.logger.Warn("websocket write failed, unregistering connection", zap.Error(err), zap.String("shop_id", shopID))
				}
				h.Unregister(shopID, c)
			}
		}(conn)
	}
}
