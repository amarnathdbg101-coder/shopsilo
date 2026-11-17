package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"shopMe/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
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

// extractClaims attempts to resolve JWT claims from Context, Query params, or WebSocket protocols.
func (c *WebSocketController) extractClaims(r *http.Request) *middleware.Claims {
	claims := middleware.GetUserFromContext(r.Context())
	if claims != nil && claims.UserID != "" {
		return claims
	}

	tokenStr := strings.TrimSpace(r.URL.Query().Get("token"))
	if tokenStr == "" {
		tokenStr = strings.TrimSpace(r.URL.Query().Get("access_token"))
	}
	if tokenStr == "" {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			tokenStr = strings.TrimSpace(authHeader[7:])
		}
	}
	if tokenStr == "" {
		proto := r.Header.Get("Sec-WebSocket-Protocol")
		if proto != "" {
			parts := strings.Split(proto, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "bearer.") || strings.HasPrefix(p, "token.") {
					tokenStr = strings.TrimPrefix(strings.TrimPrefix(p, "bearer."), "token.")
					break
				}
			}
		}
	}

	if tokenStr == "" {
		return nil
	}

	jwtSecret := utils.MustLoad().Jwt
	parsedClaims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, parsedClaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil
	}
	return parsedClaims
}

// ServeShopWebSocket handles real-time counter WebSocket connection (Protected - Shop Owner & Counter Staff)
func (c *WebSocketController) ServeShopWebSocket(w http.ResponseWriter, r *http.Request) {
	claims := c.extractClaims(r)
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil || shop == nil {
		reuse.Error(w, http.StatusNotFound, "shop not found")
		return
	}

	var responseHeader http.Header
	if subproto := r.Header.Get("Sec-WebSocket-Protocol"); subproto != "" {
		responseHeader = http.Header{}
		parts := strings.Split(subproto, ",")
		if len(parts) > 0 {
			responseHeader.Set("Sec-WebSocket-Protocol", strings.TrimSpace(parts[0]))
		}
	}

	conn, err := upgrader.Upgrade(w, r, responseHeader)
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
	claims := c.extractClaims(r)
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var responseHeader http.Header
	if subproto := r.Header.Get("Sec-WebSocket-Protocol"); subproto != "" {
		responseHeader = http.Header{}
		parts := strings.Split(subproto, ",")
		if len(parts) > 0 {
			responseHeader.Set("Sec-WebSocket-Protocol", strings.TrimSpace(parts[0]))
		}
	}

	conn, err := upgrader.Upgrade(w, r, responseHeader)
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
