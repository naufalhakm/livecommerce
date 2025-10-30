package main

import (
	"log"
	"net/http"

	"live-shopping-ai/backend/internal/api"
	"live-shopping-ai/backend/internal/db"
	"live-shopping-ai/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize database
	db.InitDatabase()

	// Initialize WebSocket hub
	hub := services.NewHub()
	go hub.Run()

	// Initialize handlers
	productHandler := api.NewProductHandler()
	streamHandler := api.NewStreamHandler(hub)

	// Setup Gin router
	r := gin.Default()

	// CORS middleware
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

	// API routes
	api := r.Group("/api")
	{
		// Product routes
		api.GET("/products", productHandler.GetProducts)
		api.POST("/products", productHandler.CreateProduct)
		api.GET("/products/:id", productHandler.GetProduct)
		api.PUT("/products/:id", productHandler.UpdateProduct)
		api.DELETE("/products/:id", productHandler.DeleteProduct)
		api.POST("/products/:id/images", productHandler.AddProductImages)
		api.POST("/products/:id/train", productHandler.TrainProduct)
		api.POST("/products/:id/predict", productHandler.PredictProduct)
		api.POST("/products/:id/pin", productHandler.PinProduct)
		api.DELETE("/products/:id/unpin", productHandler.UnpinProduct)
		api.GET("/products/pinned/:seller_id", productHandler.GetPinnedProducts)
		api.DELETE("/products/unpin-all/:seller_id", productHandler.UnpinAllProducts)
		api.GET("/training-status/:seller_id", productHandler.GetTrainingStatus)
		api.POST("/train", productHandler.TrainAllSellers)
		api.POST("/stream/predict", streamHandler.PredictFrame)

		// Stream routes
		// api.GET("/streams", streamHandler.GetLiveStreams)
		// api.POST("/streams", streamHandler.CreateLiveStream)
		// api.PUT("/streams/:id/status", streamHandler.UpdateStreamStatus)
		api.POST("/stream/process-frame", streamHandler.ProcessStreamFrame)
	}

	// Static file serving for uploads
	r.Static("/uploads", "./uploads")

	// WebSocket route
	r.GET("/ws/livestream", streamHandler.HandleWebSocket)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}