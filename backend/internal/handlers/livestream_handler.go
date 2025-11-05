package handlers

import (
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LiveStreamHandler struct {
	liveStreamService services.LiveStreamService
}

func NewLiveStreamHandler(liveStreamService services.LiveStreamService) *LiveStreamHandler {
	return &LiveStreamHandler{
		liveStreamService: liveStreamService,
	}
}

func (h *LiveStreamHandler) StartLiveStream(c *gin.Context) {
	var req entities.LiveStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, entities.LiveStreamResponse{
			Success: false,
			Message: "Invalid request data: " + err.Error(),
		})
		return
	}

	stream, err := h.liveStreamService.StartLiveStream(&req)
	if err != nil {
		c.JSON(http.StatusConflict, entities.LiveStreamResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, entities.LiveStreamResponse{
		Success: true,
		Data:    stream,
		Message: "Livestream started successfully",
	})
}

func (h *LiveStreamHandler) EndLiveStream(c *gin.Context) {
	sellerID := c.Param("seller_id")
	if sellerID == "" {
		c.JSON(http.StatusBadRequest, entities.LiveStreamResponse{
			Success: false,
			Message: "Seller ID is required",
		})
		return
	}

	err := h.liveStreamService.EndLiveStream(sellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, entities.LiveStreamResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, entities.LiveStreamResponse{
		Success: true,
		Message: "Livestream ended successfully",
	})
}

func (h *LiveStreamHandler) GetActiveLiveStreams(c *gin.Context) {
	streams, err := h.liveStreamService.GetActiveLiveStreams()
	if err != nil {
		c.JSON(http.StatusInternalServerError, entities.LiveStreamListResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, entities.LiveStreamListResponse{
		Success: true,
		Data:    streams,
		Message: "Active livestreams retrieved successfully",
	})
}

func (h *LiveStreamHandler) GetLiveStreamBySellerID(c *gin.Context) {
	sellerID := c.Param("seller_id")
	if sellerID == "" {
		c.JSON(http.StatusBadRequest, entities.LiveStreamResponse{
			Success: false,
			Message: "Seller ID is required",
		})
		return
	}

	stream, err := h.liveStreamService.GetLiveStreamBySellerID(sellerID)
	if err != nil {
		c.JSON(http.StatusNotFound, entities.LiveStreamResponse{
			Success: false,
			Message: "No active livestream found for this seller",
		})
		return
	}

	c.JSON(http.StatusOK, entities.LiveStreamResponse{
		Success: true,
		Data:    stream,
		Message: "Livestream retrieved successfully",
	})
}