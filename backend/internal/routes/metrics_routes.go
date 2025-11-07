package routes

import (
	"live-shopping-ai/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func SetupMetricsRoutes(router *gin.Engine, metricsHandler *handlers.MetricsHandler) {
	
	api := router.Group("/api")
	{
		metrics := api.Group("/metrics")
		{
			metrics.GET("/", metricsHandler.GetMetrics)
			metrics.GET("/thesis", metricsHandler.GetThesisData)
			metrics.GET("/download", metricsHandler.DownloadMetricsCSV)
		}
	}
}

func SetupExportRoutes(router *gin.Engine, exportHandler *handlers.ExportHandler) {
	api := router.Group("/api")
	{
		export := api.Group("/export")
		{
			export.POST("/thesis-results", exportHandler.GenerateThesisResults)
			export.GET("/thesis-file/:filename", exportHandler.DownloadThesisFile)
		}
	}
}