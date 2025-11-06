package routes

import (
	"live-shopping-ai/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterStreamRoutes(r *gin.Engine, streamHandler *handlers.StreamHandler) {
	api := r.Group("/api/stream")
	{
		api.POST("/process-frame", streamHandler.ProcessStreamFrame)
		api.POST("/predict", streamHandler.PredictFrame)
	}
}