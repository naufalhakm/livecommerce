package repositories

import "live-shopping-ai/backend/internal/domain/entities"

type LiveStreamRepository interface {
	CreateLiveStream(stream *entities.LiveStream) error
	GetLiveStreamBySellerID(sellerID string) (*entities.LiveStream, error)
	GetActiveLiveStreams() ([]entities.LiveStream, error)
	UpdateLiveStreamStatus(sellerID string, isLive bool) error
	UpdateViewerCount(sellerID string, count int) error
	EndLiveStream(sellerID string) error
}