package controller

import (
	"net/http"
	"os"
	"strings"
	"time"

	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Allow native mobile apps and non-browser clients
		}
		allowedEnv := os.Getenv("ALLOWED_ORIGINS")
		if allowedEnv == "" || allowedEnv == "*" {
			return true // Permissive mode for local development
		}
		for _, allowed := range strings.Split(allowedEnv, ",") {
			if strings.EqualFold(strings.TrimSpace(origin), strings.TrimSpace(allowed)) {
				return true
			}
		}
		return false
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

// ServeShopWebSocket handles real-time counter WebSocket connection (Protected - Shop Owner)
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

	c.wsHub.Register(shop.ID, conn)

	// Keep-Alive Ping/Pong Loop
	go func() {
		defer c.wsHub.Unregister(shop.ID, conn)

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
