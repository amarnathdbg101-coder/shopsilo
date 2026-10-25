package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WSEvent struct {
	Event     string      `json:"event"` // 'NEW_RESERVATION', 'PRODUCT_MUTATED', 'STOCK_CHANGED', 'POS_SALE_COMPLETED', 'SHOP_STATUS_CHANGED', etc.
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
}

// SafeConn wraps a gorilla websocket.Conn with a dedicated write mutex to eliminate concurrent write panics
type SafeConn struct {
	Conn    *websocket.Conn
	writeMu sync.Mutex
}

func NewSafeConn(conn *websocket.Conn) *SafeConn {
	return &SafeConn{Conn: conn}
}

func (s *SafeConn) WriteMessage(messageType int, data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.Conn.WriteMessage(messageType, data)
}

func (s *SafeConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.Conn.WriteControl(messageType, data, deadline)
}

func (s *SafeConn) Close() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.Conn.Close()
}

type WebSocketHub struct {
	mu                  sync.RWMutex
	shopConnections     map[string][]*SafeConn // map[shopID][]*SafeConn
	storefrontConns     map[string][]*SafeConn // map[shopSlug][]*SafeConn (customers viewing storefront)
	customerConnections map[string][]*SafeConn // map[customerID][]*SafeConn (customers tracking personal orders)
	logger              *zap.Logger
}

var (
	globalWSHub *WebSocketHub
	wsHubOnce   sync.Once
)

func GetWebSocketHub(logger *zap.Logger) *WebSocketHub {
	wsHubOnce.Do(func() {
		globalWSHub = &WebSocketHub{
			shopConnections:     make(map[string][]*SafeConn),
			storefrontConns:     make(map[string][]*SafeConn),
			customerConnections: make(map[string][]*SafeConn),
			logger:              logger,
		}
	})
	return globalWSHub
}

// Register connects a merchant shop counter device to shopID channel
func (h *WebSocketHub) Register(shopID string, conn *SafeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.shopConnections[shopID] = append(h.shopConnections[shopID], conn)
	if h.logger != nil {
		h.logger.Info("websocket client connected to shop channel", zap.String("shop_id", shopID), zap.Int("total_connections", len(h.shopConnections[shopID])))
	}
}

func (h *WebSocketHub) Unregister(shopID string, conn *SafeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns := h.shopConnections[shopID]
	for i, c := range conns {
		if c == conn || (c != nil && conn != nil && c.Conn == conn.Conn) {
			_ = c.Close()
			h.shopConnections[shopID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}

	if len(h.shopConnections[shopID]) == 0 {
		delete(h.shopConnections, shopID)
	}
}

// RegisterStorefront connects a customer browser session to a specific shop's live storefront channel
func (h *WebSocketHub) RegisterStorefront(shopSlug string, conn *SafeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.storefrontConns[shopSlug] = append(h.storefrontConns[shopSlug], conn)
	if h.logger != nil {
		h.logger.Info("customer websocket connected to storefront channel", zap.String("shop_slug", shopSlug), zap.Int("total_viewers", len(h.storefrontConns[shopSlug])))
	}
}

func (h *WebSocketHub) UnregisterStorefront(shopSlug string, conn *SafeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns := h.storefrontConns[shopSlug]
	for i, c := range conns {
		if c == conn || (c != nil && conn != nil && c.Conn == conn.Conn) {
			_ = c.Close()
			h.storefrontConns[shopSlug] = append(conns[:i], conns[i+1:]...)
			break
		}
	}

	if len(h.storefrontConns[shopSlug]) == 0 {
		delete(h.storefrontConns, shopSlug)
	}
}

// RegisterCustomer connects an authenticated customer device to their private notification channel
func (h *WebSocketHub) RegisterCustomer(customerID string, conn *SafeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.customerConnections[customerID] = append(h.customerConnections[customerID], conn)
	if h.logger != nil {
		h.logger.Info("customer websocket connected to personal channel", zap.String("customer_id", customerID))
	}
}

func (h *WebSocketHub) UnregisterCustomer(customerID string, conn *SafeConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns := h.customerConnections[customerID]
	for i, c := range conns {
		if c == conn || (c != nil && conn != nil && c.Conn == conn.Conn) {
			_ = c.Close()
			h.customerConnections[customerID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}

	if len(h.customerConnections[customerID]) == 0 {
		delete(h.customerConnections, customerID)
	}
}

// BroadcastToShop sends real-time JSON alert to all active counter devices of that shop in 0ms!
func (h *WebSocketHub) BroadcastToShop(shopID string, eventType string, payload interface{}) {
	h.mu.RLock()
	conns, exists := h.shopConnections[shopID]
	if !exists || len(conns) == 0 {
		h.mu.RUnlock()
		return
	}
	activeConns := make([]*SafeConn, len(conns))
	copy(activeConns, conns)
	h.mu.RUnlock()

	eventBytes := h.formatEvent(eventType, payload)
	if eventBytes == nil {
		return
	}

	for _, safe := range activeConns {
		go func(sc *SafeConn) {
			if err := sc.WriteMessage(websocket.TextMessage, eventBytes); err != nil {
				h.Unregister(shopID, sc)
			}
		}(safe)
	}
}

// BroadcastToStorefront sends real-time updates to all active customers currently browsing that shop's catalog
func (h *WebSocketHub) BroadcastToStorefront(shopSlug string, eventType string, payload interface{}) {
	h.mu.RLock()
	conns, exists := h.storefrontConns[shopSlug]
	if !exists || len(conns) == 0 {
		h.mu.RUnlock()
		return
	}
	activeConns := make([]*SafeConn, len(conns))
	copy(activeConns, conns)
	h.mu.RUnlock()

	eventBytes := h.formatEvent(eventType, payload)
	if eventBytes == nil {
		return
	}

	for _, safe := range activeConns {
		go func(sc *SafeConn) {
			if err := sc.WriteMessage(websocket.TextMessage, eventBytes); err != nil {
				h.UnregisterStorefront(shopSlug, sc)
			}
		}(safe)
	}
}

// BroadcastToCustomer sends real-time order/reservation notification to specific customer
func (h *WebSocketHub) BroadcastToCustomer(customerID string, eventType string, payload interface{}) {
	h.mu.RLock()
	conns, exists := h.customerConnections[customerID]
	if !exists || len(conns) == 0 {
		h.mu.RUnlock()
		return
	}
	activeConns := make([]*SafeConn, len(conns))
	copy(activeConns, conns)
	h.mu.RUnlock()

	eventBytes := h.formatEvent(eventType, payload)
	if eventBytes == nil {
		return
	}

	for _, safe := range activeConns {
		go func(sc *SafeConn) {
			if err := sc.WriteMessage(websocket.TextMessage, eventBytes); err != nil {
				h.UnregisterCustomer(customerID, sc)
			}
		}(safe)
	}
}

// BroadcastToShopAndStorefront broadcasts to both merchant counter and public storefront viewers
func (h *WebSocketHub) BroadcastToShopAndStorefront(shopID string, shopSlug string, eventType string, payload interface{}) {
	h.BroadcastToShop(shopID, eventType, payload)
	if shopSlug != "" {
		h.BroadcastToStorefront(shopSlug, eventType, payload)
	}
}

func (h *WebSocketHub) formatEvent(eventType string, payload interface{}) []byte {
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
		return nil
	}
	return eventBytes
}
