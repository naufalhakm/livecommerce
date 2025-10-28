package api

import (
	"net/http"

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

	// Broadcast detection results to seller's room
	if len(result.Predictions) > 0 {
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