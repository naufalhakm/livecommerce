package api

import (
	"context"
	"net/http"

	"live-shopping-ai/backend/internal/db"
	"live-shopping-ai/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type StreamHandler struct {
	hub      *services.Hub
	mlClient *services.MLClient
}

func NewStreamHandler(hub *services.Hub) *StreamHandler {
	return &StreamHandler{
		hub:      hub,
		mlClient: services.NewMLClient(),
	}
}

func (h *StreamHandler) HandleWebSocket(c *gin.Context) {
	services.HandleWebSocket(h.hub, c.Writer, c.Request)
}

// func (h *StreamHandler) CreateLiveStream(c *gin.Context) {
// 	var stream models.LiveStream
// 	if err := c.ShouldBindJSON(&stream); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	query := `
// 		INSERT INTO live_streams (title, seller_id, is_active)
// 		VALUES ($1, $2, $3)
// 		RETURNING id, created_at, updated_at
// 	`
	
// 	err := db.DB.QueryRow(
// 		context.Background(), query,
// 		stream.Title, stream.SellerID, stream.IsActive,
// 	).Scan(&stream.ID, &stream.CreatedAt, &stream.UpdatedAt)
	
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusCreated, stream)
// }

// func (h *StreamHandler) GetLiveStreams(c *gin.Context) {
// 	query := `
// 		SELECT ls.id, ls.title, ls.seller_id, ls.is_active, ls.created_at, ls.updated_at,
// 		       s.id, s.name, s.email, s.created_at, s.updated_at
// 		FROM live_streams ls
// 		LEFT JOIN sellers s ON ls.seller_id = s.id
// 		ORDER BY ls.created_at DESC
// 	`
	
// 	rows, err := db.DB.Query(context.Background(), query)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	defer rows.Close()

// 	streams := make([]models.LiveStream, 0)
// 	for rows.Next() {
// 		var ls models.LiveStream
// 		var s models.Seller
		
// 		err := rows.Scan(
// 			&ls.ID, &ls.Title, &ls.SellerID, &ls.IsActive, &ls.CreatedAt, &ls.UpdatedAt,
// 			&s.ID, &s.Name, &s.Email, &s.CreatedAt, &s.UpdatedAt,
// 		)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 			return
// 		}
		
// 		ls.Seller = &s
// 		streams = append(streams, ls)
// 	}

// 	c.JSON(http.StatusOK, streams)
// }

// func (h *StreamHandler) UpdateStreamStatus(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stream ID"})
// 		return
// 	}

// 	var updateData struct {
// 		IsActive bool `json:"is_active"`
// 	}
// 	if err := c.ShouldBindJSON(&updateData); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	query := `
// 		UPDATE live_streams 
// 		SET is_active = $1, updated_at = CURRENT_TIMESTAMP
// 		WHERE id = $2
// 		RETURNING title, seller_id, created_at, updated_at
// 	`
	
// 	var stream models.LiveStream
// 	err = db.DB.QueryRow(context.Background(), query, updateData.IsActive, id).Scan(
// 		&stream.Title, &stream.SellerID, &stream.CreatedAt, &stream.UpdatedAt,
// 	)
	
// 	if err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Stream not found"})
// 		return
// 	}

// 	stream.ID = id
// 	stream.IsActive = updateData.IsActive

// 	// Broadcast stream status change to seller's room
// 	roomID := strconv.Itoa(stream.SellerID)
// 	h.hub.BroadcastToRoom(roomID, services.Message{
// 		Type: "stream_status_change",
// 		Data: map[string]interface{}{
// 			"stream_id": stream.ID,
// 			"is_active": stream.IsActive,
// 		},
// 	})

// 	c.JSON(http.StatusOK, stream)
// }

func (h *StreamHandler) ProcessStreamFrame(c *gin.Context) {
	sellerID := c.Query("seller_id")
	if sellerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller_id is required"})
		return
	}

	file, err := c.FormFile("frame")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No frame file provided"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open frame file"})
		return
	}
	defer src.Close()

	frameData := make([]byte, file.Size)
	src.Read(frameData)

	// Send frame to ML service for prediction
	result, err := h.mlClient.PredictProduct(sellerID, frameData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Auto pin/unpin products based on predictions
	if len(result.Predictions) > 0 {
		for _, prediction := range result.Predictions {
			if prediction.SimilarityScore > 0.8 { // High confidence threshold
				// Auto pin product
				h.autoPinProduct(prediction.ProductID, sellerID, prediction.SimilarityScore)
			}
		}
		
		// Broadcast detection results to seller's room
		h.hub.BroadcastToRoom(sellerID, services.Message{
			Type: "product_detection",
			Data: map[string]interface{}{
				"seller_id":   sellerID,
				"predictions": result.Predictions,
			},
		})
	}

	c.JSON(http.StatusOK, result)
}

func (h *StreamHandler) PredictFrame(c *gin.Context) {
	sellerID := c.Query("seller_id")
	if sellerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "seller_id is required"})
		return
	}

	file, err := c.FormFile("frame")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No frame file provided"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open frame file"})
		return
	}
	defer src.Close()

	frameData := make([]byte, file.Size)
	src.Read(frameData)

	// Call ML service for prediction
	result, err := h.mlClient.PredictProduct(sellerID, frameData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Auto pin/unpin products based on predictions
	if len(result.Predictions) > 0 {
		for _, prediction := range result.Predictions {
			if prediction.SimilarityScore > 0.8 { // High confidence threshold
				// Auto pin product
				h.autoPinProduct(prediction.ProductID, sellerID, prediction.SimilarityScore)
			}
		}
		
		// Broadcast detection results to seller's room
		h.hub.BroadcastToRoom(sellerID, services.Message{
			Type: "product_detection",
			Data: map[string]interface{}{
				"seller_id":   sellerID,
				"predictions": result.Predictions,
			},
		})
	}

	c.JSON(http.StatusOK, result)
}

func (h *StreamHandler) autoPinProduct(productID, sellerID string, similarityScore float64) {
	// Convert productID from string to int
	var dbProductID int
	query := `SELECT id FROM products WHERE id = $1 AND seller_id = $2 LIMIT 1`
	err := db.DB.QueryRow(context.Background(), query, productID, sellerID).Scan(&dbProductID)
	if err != nil {
		logger.Printf("Product not found for auto-pin: %s", productID)
		return
	}

	// Unpin previous products
	unpinQuery := `UPDATE pinned_products SET is_pinned = false WHERE seller_id = $1 AND is_pinned = true`
	db.DB.Exec(context.Background(), unpinQuery, sellerID)

	// Pin new product
	pinQuery := `
		INSERT INTO pinned_products (product_id, seller_id, similarity_score, is_pinned, pinned_at)
		VALUES ($1, $2, $3, true, CURRENT_TIMESTAMP)
		ON CONFLICT (product_id, seller_id) 
		DO UPDATE SET similarity_score = $3, is_pinned = true, pinned_at = CURRENT_TIMESTAMP
	`
	db.DB.Exec(context.Background(), pinQuery, dbProductID, sellerID, similarityScore)

	// Broadcast pin update
	h.hub.BroadcastToRoom(sellerID, services.Message{
		Type: "product_pinned",
		Data: map[string]interface{}{
			"product_id":       dbProductID,
			"similarity_score": similarityScore,
		},
	})
}