package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"live-shopping-ai/backend/internal/db"
	"live-shopping-ai/backend/internal/models"
	"live-shopping-ai/backend/internal/services"

	"github.com/gin-gonic/gin"
)

var logger = log.New(os.Stdout, "[PRODUCT] ", log.LstdFlags)

type ProductHandler struct {
	mlClient *services.MLClient
}

func NewProductHandler() *ProductHandler {
	return &ProductHandler{
		mlClient: services.NewMLClient(),
	}
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
	query := `
		SELECT p.id, p.name, p.description, p.price, p.seller_id, p.created_at, p.updated_at,
		       COALESCE(array_agg(i.image_url) FILTER (WHERE i.image_url IS NOT NULL), '{}') as image_urls
		FROM products p
		LEFT JOIN images i ON p.id = i.product_id
		GROUP BY p.id, p.name, p.description, p.price, p.seller_id, p.created_at, p.updated_at
		ORDER BY p.created_at DESC
	`
	
	rows, err := db.DB.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		var p models.Product
		var imageURLs []string
		
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.SellerID, &p.CreatedAt, &p.UpdatedAt, &imageURLs,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		// Convert image URLs to Image structs
		for _, url := range imageURLs {
			if url != "" {
				p.Images = append(p.Images, models.Image{ImageURL: url})
			}
		}
		
		products = append(products, p)
	}

	c.JSON(http.StatusOK, products)
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var product models.Product
	
	// Handle multipart form data
	product.Name = c.PostForm("name")
	product.Description = c.PostForm("description")
	priceStr := c.PostForm("price")
	sellerIDStr := c.PostForm("seller_id")
	
	// Debug logging
	c.Header("Content-Type", "application/json")
	if product.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required", "received_name": product.Name})
		return
	}
	if priceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Price is required", "received_price": priceStr})
		return
	}
	
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price"})
		return
	}
	product.Price = price
	
	// Default seller_id if not provided
	sellerID := 1
	if sellerIDStr != "" {
		sellerID, err = strconv.Atoi(sellerIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid seller_id"})
			return
		}
	}
	product.SellerID = sellerID

	// Create product first
	query := `
		INSERT INTO products (name, description, price, seller_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	
	err = db.DB.QueryRow(
		context.Background(), query,
		product.Name, product.Description, product.Price, product.SellerID,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create ML dataset structure
	err = h.createMLDataset(product.ID, product.SellerID, product.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ML dataset structure", "details": err.Error()})
		return
	}

	// Handle multiple file uploads
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form", "details": err.Error()})
		return
	}
	
	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "Product created without images", "product": product})
		return
	}
	
	for i, file := range files {
		storageService := services.NewStorageService()
		imageURL, err := storageService.UploadFromForm(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload image %d", i), "details": err.Error()})
			return
		}
		
		// Save to ML dataset
		err = h.saveToMLDataset(product.ID, product.SellerID, file)
		if err != nil {
			logger.Printf("Warning: Failed to save to ML dataset: %v", err)
		}
		
		// Insert image
		imageQuery := `INSERT INTO images (product_id, image_url) VALUES ($1, $2)`
		_, err = db.DB.Exec(context.Background(), imageQuery, product.ID, imageURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save image %d to database", i), "details": err.Error()})
			return
		}
		
		product.Images = append(product.Images, models.Image{
			ProductID: product.ID,
			ImageURL:  imageURL,
		})
	}

	c.JSON(http.StatusCreated, product)
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	query := `
		SELECT p.id, p.name, p.description, p.price, p.seller_id, p.created_at, p.updated_at
		FROM products p
		WHERE p.id = $1
	`
	
	var p models.Product
	
	err = db.DB.QueryRow(context.Background(), query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.SellerID, &p.CreatedAt, &p.UpdatedAt,
	)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	
	// Get images
	imageQuery := `SELECT id, image_url FROM images WHERE product_id = $1`
	imageRows, err := db.DB.Query(context.Background(), imageQuery, id)
	if err == nil {
		defer imageRows.Close()
		for imageRows.Next() {
			var img models.Image
			imageRows.Scan(&img.ID, &img.ImageURL)
			img.ProductID = id
			p.Images = append(p.Images, img)
		}
	}
	
	c.JSON(http.StatusOK, p)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		UPDATE products 
		SET name = $1, description = $2, price = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
		RETURNING updated_at
	`
	
	err = db.DB.QueryRow(
		context.Background(), query,
		product.Name, product.Description, product.Price, id,
	).Scan(&product.UpdatedAt)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	product.ID = id
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) AddProductImages(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Check if product exists
	var exists bool
	err = db.DB.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", id).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Handle multiple file uploads
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No images provided"})
		return
	}

	var addedImages []models.Image
	for i, file := range files {
		storageService := services.NewStorageService()
		imageURL, err := storageService.UploadFromForm(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload image %d", i), "details": err.Error()})
			return
		}

		// Insert image
		imageQuery := `INSERT INTO images (product_id, image_url) VALUES ($1, $2) RETURNING id`
		var imageID int
		err = db.DB.QueryRow(context.Background(), imageQuery, id, imageURL).Scan(&imageID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save image %d to database", i), "details": err.Error()})
			return
		}
		
		addedImages = append(addedImages, models.Image{
			ID:        imageID,
			ProductID: id,
			ImageURL:  imageURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Images added successfully", "images": addedImages})
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	query := `DELETE FROM products WHERE id = $1`
	_, err = db.DB.Exec(context.Background(), query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

func (h *ProductHandler) TrainProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	query := `SELECT seller_id FROM products WHERE id = $1`
	var sellerID int
	err = db.DB.QueryRow(context.Background(), query, id).Scan(&sellerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	result, err := h.mlClient.TrainModel(strconv.Itoa(sellerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ProductHandler) GetTrainingStatus(c *gin.Context) {
	sellerID := c.Param("seller_id")
	if sellerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller_id is required"})
		return
	}

	status, err := h.mlClient.GetTrainingStatus(sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *ProductHandler) createMLDataset(productID, sellerID int, productName string) error {
	// Create ML dataset directory structure
	mlDir := fmt.Sprintf("../ml_service/datasets/seller_%d/product_%d", sellerID, productID)
	imagesDir := filepath.Join(mlDir, "images")
	
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return err
	}
	
	// Create metadata.json
	metadata := map[string]interface{}{
		"product_id":   fmt.Sprintf("product_%d", productID),
		"product_name": productName,
		"seller_id":    fmt.Sprintf("seller_%d", sellerID),
	}
	
	metadataPath := filepath.Join(mlDir, "metadata.json")
	file, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return json.NewEncoder(file).Encode(metadata)
}

func (h *ProductHandler) saveToMLDataset(productID, sellerID int, file *multipart.FileHeader) error {
	// Save image to ML dataset
	mlDir := fmt.Sprintf("../ml_service/datasets/seller_%d/product_%d/images", sellerID, productID)
	filePath := filepath.Join(mlDir, file.Filename)
	
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()
	
	_, err = io.Copy(dst, src)
	return err
}

func (h *ProductHandler) PredictProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	query := `SELECT seller_id FROM products WHERE id = $1`
	var sellerID int
	err = db.DB.QueryRow(context.Background(), query, id).Scan(&sellerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open image file"})
		return
	}
	defer src.Close()

	imageData := make([]byte, file.Size)
	src.Read(imageData)

	result, err := h.mlClient.PredictProduct(strconv.Itoa(sellerID), imageData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ProductHandler) PinProduct(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var pinData struct {
		SellerID        int     `json:"seller_id"`
		SimilarityScore float64 `json:"similarity_score"`
	}
	if err := c.ShouldBindJSON(&pinData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Unpin previous product for this seller
	unpinQuery := `UPDATE pinned_products SET is_pinned = false WHERE seller_id = $1 AND is_pinned = true`
	_, err = db.DB.Exec(context.Background(), unpinQuery, pinData.SellerID)
	if err != nil {
		logger.Printf("Warning: Failed to unpin previous products: %v", err)
	}

	// Pin new product
	pinQuery := `
		INSERT INTO pinned_products (product_id, seller_id, similarity_score, is_pinned, pinned_at)
		VALUES ($1, $2, $3, true, CURRENT_TIMESTAMP)
		ON CONFLICT (product_id, seller_id) 
		DO UPDATE SET similarity_score = $3, is_pinned = true, pinned_at = CURRENT_TIMESTAMP
		RETURNING id
	`
	
	var pinnedID int
	err = db.DB.QueryRow(context.Background(), pinQuery, productID, pinData.SellerID, pinData.SimilarityScore).Scan(&pinnedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product pinned successfully", "pinned_id": pinnedID})
}

func (h *ProductHandler) UnpinProduct(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	sellerID := c.Query("seller_id")
	if sellerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller_id is required"})
		return
	}

	unpinQuery := `UPDATE pinned_products SET is_pinned = false WHERE product_id = $1 AND seller_id = $2`
	_, err = db.DB.Exec(context.Background(), unpinQuery, productID, sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product unpinned successfully"})
}

func (h *ProductHandler) GetPinnedProducts(c *gin.Context) {
	sellerID := c.Param("seller_id")

	query := `
		SELECT pp.id, pp.product_id, pp.seller_id, pp.similarity_score, pp.is_pinned, pp.pinned_at,
		       p.name, p.description, p.price
		FROM pinned_products pp
		JOIN products p ON pp.product_id = p.id
		WHERE pp.seller_id = $1 AND pp.is_pinned = true
		ORDER BY pp.pinned_at DESC
	`
	
	rows, err := db.DB.Query(context.Background(), query, sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	pinnedProducts := make([]models.PinnedProduct, 0)
	for rows.Next() {
		var pp models.PinnedProduct
		var p models.Product
		
		err := rows.Scan(
			&pp.ID, &pp.ProductID, &pp.SellerID, &pp.SimilarityScore, &pp.IsPinned, &pp.PinnedAt,
			&p.Name, &p.Description, &p.Price,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		p.ID = pp.ProductID
		pp.Product = &p
		pinnedProducts = append(pinnedProducts, pp)
	}

	c.JSON(http.StatusOK, pinnedProducts)
}

func (h *ProductHandler) TrainAllSellers(c *gin.Context) {
	sellerID := c.Query("seller_id")
	if sellerID == "" {
		sellerID = "1" // default
	}

	result, err := h.mlClient.TrainModel(sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}