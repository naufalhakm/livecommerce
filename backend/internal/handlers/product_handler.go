package handlers

import (
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	productService services.ProductService
}

func NewProductHandler(productService services.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
	products, err := h.productService.GetProducts(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, products)
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := h.productService.GetProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Product not found"})
		return
	}
	c.JSON(200, product)
}

func (h *ProductHandler) GetProductsBySeller(c *gin.Context) {
	sellerID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid seller ID"})
		return
	}

	products, err := h.productService.GetProductsBySellerID(c.Request.Context(), sellerID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, products)
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var product entities.Product
	
	product.Name = c.PostForm("name")
	product.Description = c.PostForm("description")
	priceStr := c.PostForm("price")
	sellerIDStr := c.PostForm("seller_id")
	
	if product.Name == "" {
		c.JSON(400, gin.H{"error": "Name is required"})
		return
	}
	if priceStr == "" {
		c.JSON(400, gin.H{"error": "Price is required"})
		return
	}
	
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid price"})
		return
	}
	product.Price = price
	
	sellerID := 1
	if sellerIDStr != "" {
		sellerID, err = strconv.Atoi(sellerIDStr)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid seller_id"})
			return
		}
	}
	product.SellerID = sellerID

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to parse multipart form"})
		return
	}
	
	files := form.File["images"]
	
	if err := h.productService.CreateProduct(c.Request.Context(), &product, files); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, product)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid product ID"})
		return
	}

	var product entities.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	product.ID = id
	if err := h.productService.UpdateProduct(c.Request.Context(), &product); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, product)
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid product ID"})
		return
	}

	if err := h.productService.DeleteProduct(c.Request.Context(), id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Product deleted successfully"})
}

func (h *ProductHandler) AddProductImages(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid product ID"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(400, gin.H{"error": "No images provided"})
		return
	}

	addedImages, err := h.productService.AddProductImages(c.Request.Context(), id, files)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Images added successfully", "images": addedImages})
}

func (h *ProductHandler) TrainProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid product ID"})
		return
	}

	result, err := h.productService.TrainProductModel(c.Request.Context(), id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

func (h *ProductHandler) PredictProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid product ID"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(400, gin.H{"error": "No image file provided"})
		return
	}

	result, err := h.productService.PredictProduct(c.Request.Context(), id, file)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

func (h *ProductHandler) PinProduct(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid product ID"})
		return
	}

	var pinData struct {
		SellerID        int     `json:"seller_id"`
		SimilarityScore float64 `json:"similarity_score"`
	}
	if err := c.ShouldBindJSON(&pinData); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := h.productService.PinProduct(c.Request.Context(), productID, pinData.SellerID, pinData.SimilarityScore); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Product pinned successfully"})
}

func (h *ProductHandler) UnpinProduct(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid product ID"})
		return
	}

	sellerID := c.Query("seller_id")
	if sellerID == "" {
		c.JSON(400, gin.H{"error": "seller_id is required"})
		return
	}

	if err := h.productService.UnpinProduct(c.Request.Context(), productID, sellerID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Product unpinned successfully"})
}

func (h *ProductHandler) GetPinnedProducts(c *gin.Context) {
	sellerID := c.Param("seller_id")

	products, err := h.productService.GetPinnedProducts(c.Request.Context(), sellerID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, products)
}

func (h *ProductHandler) UnpinAllProducts(c *gin.Context) {
	sellerID := c.Param("seller_id")

	rowsAffected, err := h.productService.UnpinAllProducts(c.Request.Context(), sellerID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message":       "All products unpinned successfully",
		"rows_affected": rowsAffected,
	})
}

func (h *ProductHandler) TrainAllSellers(c *gin.Context) {
	sellerID := c.Query("seller_id")
	if sellerID == "" {
		sellerID = "1"
	}

	result, err := h.productService.TrainAllSellers(c.Request.Context(), sellerID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

func (h *ProductHandler) GetTrainingStatus(c *gin.Context) {
	sellerID := c.Param("seller_id")

	status, err := h.productService.GetTrainingStatus(c.Request.Context(), sellerID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, status)
}