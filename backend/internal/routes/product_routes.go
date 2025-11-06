package routes

import (
	"live-shopping-ai/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterProductRoutes(r *gin.Engine, productHandler *handlers.ProductHandler) {
	api := r.Group("/api")
	{
		api.GET("/products", productHandler.GetProducts)
		api.POST("/products", productHandler.CreateProduct)
		api.GET("/products/:id", productHandler.GetProduct)
		api.GET("/products/seller/:id", productHandler.GetProductsBySeller)
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
	}
}