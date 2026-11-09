package services

import (
	"sync"

	gowshub "github.com/amarnathdbg101-coder/go/go-wshub"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WSEvent standardizes real-time WebSocket payloads across microservices and clients.
type WSEvent = gowshub.Event

// SafeConn wraps a raw WebSocket connection with dedicated write mutex serialization and deadlines.
type SafeConn = gowshub.SafeConn

// NewSafeConn wraps a gorilla websocket.Conn with thread-safe write protection.
func NewSafeConn(conn *websocket.Conn) *SafeConn {
	return gowshub.NewSafeConn(conn)
}

// WebSocketHub manages real-time merchant counters, customer tracking, and public storefront pub-sub channels.
// Powered by go-wshub for O(1) room membership, dead connection auto-pruning, and concurrent write safety.
type WebSocketHub struct {
	hub    *gowshub.Hub
	logger *zap.Logger
}

var (
	globalWSHub *WebSocketHub
	wsHubOnce   sync.Once
)

// GetWebSocketHub returns the singleton WebSocketHub instance.
func GetWebSocketHub(logger *zap.Logger) *WebSocketHub {
	wsHubOnce.Do(func() {
		hub := gowshub.New(
			gowshub.WithOnConnect(func(room string, conn *gowshub.SafeConn) {
				if logger != nil {
					logger.Debug("websocket client joined room", zap.String("room", room))
				}
			}),
			gowshub.WithOnDisconnect(func(room string, conn *gowshub.SafeConn) {
				if logger != nil {
					logger.Debug("websocket client left room", zap.String("room", room))
				}
			}),
		)
		globalWSHub = &WebSocketHub{
			hub:    hub,
			logger: logger,
		}
	})
	return globalWSHub
}

// Hub returns the underlying gowshub.Hub instance.
func (h *WebSocketHub) Hub() *gowshub.Hub {
	return h.hub
}

// Register connects a merchant shop counter device to shopID channel.
func (h *WebSocketHub) Register(shopID string, conn *SafeConn) {
	_ = h.hub.Join("shop:"+shopID, conn)
	if h.logger != nil {
		h.logger.Info("websocket client connected to shop channel",
			zap.String("shop_id", shopID),
			zap.Int("total_connections", h.hub.RoomSize("shop:"+shopID)),
		)
	}
}

// Unregister disconnects a merchant shop counter device from shopID channel.
func (h *WebSocketHub) Unregister(shopID string, conn *SafeConn) {
	h.hub.Leave("shop:"+shopID, conn)
	_ = conn.Close()
}

// RegisterStorefront connects a customer browser session to a specific shop's live storefront channel.
func (h *WebSocketHub) RegisterStorefront(shopSlug string, conn *SafeConn) {
	_ = h.hub.Join("storefront:"+shopSlug, conn)
	if h.logger != nil {
		h.logger.Info("customer websocket connected to storefront channel",
			zap.String("shop_slug", shopSlug),
			zap.Int("total_viewers", h.hub.RoomSize("storefront:"+shopSlug)),
		)
	}
}

// UnregisterStorefront disconnects a customer browser session from storefront channel.
func (h *WebSocketHub) UnregisterStorefront(shopSlug string, conn *SafeConn) {
	h.hub.Leave("storefront:"+shopSlug, conn)
	_ = conn.Close()
}

// RegisterCustomer connects an authenticated customer device to their private notification channel.
func (h *WebSocketHub) RegisterCustomer(customerID string, conn *SafeConn) {
	_ = h.hub.Join("customer:"+customerID, conn)
	if h.logger != nil {
		h.logger.Info("customer websocket connected to personal channel", zap.String("customer_id", customerID))
	}
}

// UnregisterCustomer disconnects an authenticated customer device from their private channel.
func (h *WebSocketHub) UnregisterCustomer(customerID string, conn *SafeConn) {
	h.hub.Leave("customer:"+customerID, conn)
	_ = conn.Close()
}

// BroadcastToShop sends real-time JSON alert to all active counter devices of that shop in 0ms!
func (h *WebSocketHub) BroadcastToShop(shopID string, eventType string, payload interface{}) {
	_ = h.hub.BroadcastEvent("shop:"+shopID, eventType, payload)
}

// BroadcastToStorefront sends real-time updates to all active customers currently browsing that shop's catalog.
func (h *WebSocketHub) BroadcastToStorefront(shopSlug string, eventType string, payload interface{}) {
	_ = h.hub.BroadcastEvent("storefront:"+shopSlug, eventType, payload)
}

// BroadcastToCustomer sends real-time order/reservation notification to a specific customer.
func (h *WebSocketHub) BroadcastToCustomer(customerID string, eventType string, payload interface{}) {
	_ = h.hub.BroadcastEvent("customer:"+customerID, eventType, payload)
}

// BroadcastToShopAndStorefront broadcasts to both merchant counter and public storefront viewers.
func (h *WebSocketHub) BroadcastToShopAndStorefront(shopID string, shopSlug string, eventType string, payload interface{}) {
	h.BroadcastToShop(shopID, eventType, payload)
	if shopSlug != "" {
		h.BroadcastToStorefront(shopSlug, eventType, payload)
	}
}
