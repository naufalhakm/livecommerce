package routes

import (
	"live-shopping-ai/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupLiveStreamRoutes(router *gin.Engine, handler *handlers.LiveStreamHandler) {
	api := router.Group("/api")
	{
		livestream := api.Group("/livestreams")
		{
			livestream.POST("/start", handler.StartLiveStream)
			livestream.POST("/end/:seller_id", handler.EndLiveStream)
			livestream.GET("/active", handler.GetActiveLiveStreams)
			livestream.GET("/seller/:seller_id", handler.GetLiveStreamBySellerID)
		}
	}
}