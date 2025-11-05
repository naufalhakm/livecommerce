package handlers

import (
	"live-shopping-ai/backend/internal/domain/services"

	"github.com/gin-gonic/gin"
)

type StreamHandler struct {
	streamService services.StreamService
}

func NewStreamHandler(streamService services.StreamService) *StreamHandler {
	return &StreamHandler{
		streamService: streamService,
	}
}

func (h *StreamHandler) ProcessStreamFrame(c *gin.Context) {
	sellerID := c.Query("seller_id")
	if sellerID == "" {
		c.JSON(400, gin.H{"error": "seller_id is required"})
		return
	}

	file, err := c.FormFile("frame")
	if err != nil {
		c.JSON(400, gin.H{"error": "No frame file provided"})
		return
	}

	result, err := h.streamService.ProcessStreamFrame(c.Request.Context(), sellerID, file)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

func (h *StreamHandler) PredictFrame(c *gin.Context) {
	sellerID := c.Query("seller_id")
	if sellerID == "" {
		c.JSON(400, gin.H{"error": "seller_id is required"})
		return
	}

	file, err := c.FormFile("frame")
	if err != nil {
		c.JSON(400, gin.H{"error": "No frame file provided"})
		return
	}

	result, err := h.streamService.PredictFrame(c.Request.Context(), sellerID, file)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}