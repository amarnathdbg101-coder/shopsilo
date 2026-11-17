package controller

import (
	"net/http"
	"strings"
	"time"

	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all cross-platform clients (Flutter Mobile, Desktop, Web)
	},
}

type WebSocketController struct {
	shopService *services.ShopService
	wsHub       *services.WebSocketHub
	logger      *zap.Logger
}

func NewWebSocketController(shopService *services.ShopService, logger *zap.Logger) *WebSocketController {
	return &WebSocketController{
		shopService: shopService,
		wsHub:       services.GetWebSocketHub(logger),
		logger:      logger,
	}
}

// ServeShopWebSocket handles real-time counter WebSocket connection (Protected - Shop Owner & Counter Staff)
func (c *WebSocketController) ServeShopWebSocket(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil || shop == nil {
		reuse.Error(w, http.StatusNotFound, "shop not found")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.Error("failed to upgrade HTTP connection to websocket", zap.Error(err))
		return
	}

	safeConn := services.NewSafeConn(conn)
	c.wsHub.Register(shop.ID, safeConn)

	// Keep-Alive Ping/Pong & Heartbeat Loop
	stopPing := make(chan struct{})
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := safeConn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			case <-stopPing:
				return
			}
		}
	}()

	go func() {
		defer func() {
			close(stopPing)
			c.wsHub.Unregister(shop.ID, safeConn)
		}()

		conn.SetReadLimit(2048)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// ServePublicStorefrontWebSocket handles real-time catalog & inventory updates for customers browsing a storefront
func (c *WebSocketController) ServePublicStorefrontWebSocket(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		reuse.Error(w, http.StatusBadRequest, "shop slug is required")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.Error("failed to upgrade storefront connection to websocket", zap.Error(err), zap.String("slug", slug))
		return
	}

	safeConn := services.NewSafeConn(conn)
	c.wsHub.RegisterStorefront(slug, safeConn)

	stopPing := make(chan struct{})
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := safeConn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			case <-stopPing:
				return
			}
		}
	}()

	go func() {
		defer func() {
			close(stopPing)
			c.wsHub.UnregisterStorefront(slug, safeConn)
		}()

		conn.SetReadLimit(2048)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// ServeCustomerWebSocket handles personal order & pickup notifications for authenticated customers
func (c *WebSocketController) ServeCustomerWebSocket(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.Error("failed to upgrade customer connection to websocket", zap.Error(err))
		return
	}

	safeConn := services.NewSafeConn(conn)
	c.wsHub.RegisterCustomer(claims.UserID, safeConn)

	stopPing := make(chan struct{})
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := safeConn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			case <-stopPing:
				return
			}
		}
	}()

	go func() {
		defer func() {
			close(stopPing)
			c.wsHub.UnregisterCustomer(claims.UserID, safeConn)
		}()

		conn.SetReadLimit(2048)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
