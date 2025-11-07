package main

import (
	"log"
	"net/http"

	"live-shopping-ai/backend/internal/domain/services"
	"live-shopping-ai/backend/internal/handlers"
	"live-shopping-ai/backend/internal/infrastructure/database"
	"live-shopping-ai/backend/internal/infrastructure/mlclient"
	"live-shopping-ai/backend/internal/infrastructure/storage"
	"live-shopping-ai/backend/internal/infrastructure/webrtc"
	"live-shopping-ai/backend/internal/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
	}

	db := database.InitDatabase()

	productRepo := database.NewPostgresProductRepository(db)
	pinnedRepo := database.NewPostgresPinnedRepository(db)
	liveStreamRepo := database.NewPostgresLiveStreamRepository(db)
	mlRepo := mlclient.NewHttpMLRepository()
	storageRepo := storage.NewStorageService()
	webrtcRepo := webrtc.NewMemoryWebRTCRepository()
	mlMetricsRepo := mlclient.NewMLMetricsRepository()

	productService := services.NewProductService(productRepo, pinnedRepo, mlRepo, storageRepo)
	webrtcService := services.NewWebRTCService(webrtcRepo, liveStreamRepo)
	streamService := services.NewStreamService(mlRepo, pinnedRepo)
	liveStreamService := services.NewLiveStreamService(liveStreamRepo)
	metricsService := services.NewMetricsService(mlMetricsRepo)

	productHandler := handlers.NewProductHandler(productService)
	webrtcHandler := handlers.NewWebRTCHandler(webrtcService)
	streamHandler := handlers.NewStreamHandler(streamService)
	liveStreamHandler := handlers.NewLiveStreamHandler(liveStreamService)
	metricsHandler := handlers.NewMetricsHandler(metricsService)
	exportHandler := handlers.NewExportHandler(metricsService)

	router := setupRouter(productHandler, webrtcHandler, streamHandler, liveStreamHandler, metricsHandler, exportHandler)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func setupRouter(
	productHandler *handlers.ProductHandler,
	webrtcHandler *handlers.WebRTCHandler,
	streamHandler *handlers.StreamHandler,
	liveStreamHandler *handlers.LiveStreamHandler,
	metricsHandler *handlers.MetricsHandler,
	exportHandler *handlers.ExportHandler,
) *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	})

	routes.RegisterProductRoutes(r, productHandler)
	routes.RegisterWebRTCRoutes(r, webrtcHandler)
	routes.RegisterStreamRoutes(r, streamHandler)
	routes.SetupLiveStreamRoutes(r, liveStreamHandler)
	routes.SetupMetricsRoutes(r, metricsHandler)
	routes.SetupExportRoutes(r, exportHandler)

	r.Static("/uploads", "./uploads")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	return r
}