package handlers

import (
	"live-shopping-ai/backend/internal/domain/services"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WebRTCHandler struct {
	webrtcService services.WebRTCService
	upgrader      websocket.Upgrader
}

func NewWebRTCHandler(webrtcService services.WebRTCService) *WebRTCHandler {
	return &WebRTCHandler{
		webrtcService: webrtcService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WebRTCHandler) HandleWebRTCWebSocket(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "*")

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	if err := h.webrtcService.HandleWebSocketConnection(conn); err != nil {
	}
}

func (h *WebRTCHandler) HandleLiveStreamWebSocket(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "*")

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	if err := h.webrtcService.HandleWebSocketConnection(conn); err != nil {
	}
}

func (h *WebRTCHandler) GetWebRTCConfig(c *gin.Context) {
	serverPublicIP := os.Getenv("SERVER_PUBLIC_IP")
	if serverPublicIP == "" {
		serverPublicIP = "localhost"
	}

	config := map[string]interface{}{
		"iceServers": []map[string]interface{}{
			{"urls": "stun:stun.l.google.com:19302"},
			{"urls": "stun:stun1.l.google.com:19302"},
			{"urls": "stun:stun2.l.google.com:19302"},
			{"urls": "stun:stun3.l.google.com:19302"},
		},
		"iceTransportPolicy": "all",
		"bundlePolicy":       "max-bundle",
		"rtcpMuxPolicy":      "require",
	}

	c.JSON(200, gin.H{
		"success": true,
		"config":  config,
	})
}

func (h *WebRTCHandler) GetRoomStats(c *gin.Context) {
	roomID := c.Param("room_id")
	stats := h.webrtcService.GetRoomStats(roomID)
	
	if stats == nil {
		c.JSON(404, gin.H{"error": "Room not found"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"stats":   stats,
	})
}

func (h *WebRTCHandler) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"service":   "webrtc",
		"timestamp": time.Now().Unix(),
	})
}