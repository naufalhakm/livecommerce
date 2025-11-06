package routes

import (
	"live-shopping-ai/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterWebRTCRoutes(r *gin.Engine, webrtcHandler *handlers.WebRTCHandler) {
	api := r.Group("/api/webrtc")
	{
		api.GET("/ws", webrtcHandler.HandleWebRTCWebSocket)
		api.GET("/config", webrtcHandler.GetWebRTCConfig)
		api.GET("/health", webrtcHandler.HealthCheck)
		api.GET("/stats/:room_id", webrtcHandler.GetRoomStats)
	}
	
	r.GET("/ws/livestream", webrtcHandler.HandleLiveStreamWebSocket)
	r.GET("/ws/webrtc", webrtcHandler.HandleWebRTCWebSocket)
}