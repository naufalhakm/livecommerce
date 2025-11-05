package services

import (
	"fmt"
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/repositories"
	"time"
)

type LiveStreamService interface {
	StartLiveStream(req *entities.LiveStreamRequest) (*entities.LiveStream, error)
	EndLiveStream(sellerID string) error
	GetActiveLiveStreams() ([]entities.LiveStream, error)
	GetLiveStreamBySellerID(sellerID string) (*entities.LiveStream, error)
	UpdateViewerCount(sellerID string, count int) error
}

type liveStreamService struct {
	repo repositories.LiveStreamRepository
}

func NewLiveStreamService(repo repositories.LiveStreamRepository) LiveStreamService {
	return &liveStreamService{
		repo: repo,
	}
}

func (s *liveStreamService) StartLiveStream(req *entities.LiveStreamRequest) (*entities.LiveStream, error) {
	// Check if seller already has an active stream
	existingStream, err := s.repo.GetLiveStreamBySellerID(req.SellerID)
	if err == nil && existingStream != nil {
		return nil, fmt.Errorf("seller already has an active livestream")
	}

	stream := &entities.LiveStream{
		SellerID:    req.SellerID,
		SellerName:  req.SellerName,
		Title:       req.Title,
		Description: req.Description,
		IsLive:      true,
		ViewerCount: 0,
		StartedAt:   time.Now(),
	}

	err = s.repo.CreateLiveStream(stream)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (s *liveStreamService) EndLiveStream(sellerID string) error {
	err := s.repo.EndLiveStream(sellerID)
	if err != nil {
		return err
	}

	return nil
}

func (s *liveStreamService) GetActiveLiveStreams() ([]entities.LiveStream, error) {
	streams, err := s.repo.GetActiveLiveStreams()
	if err != nil {
		return nil, err
	}

	return streams, nil
}

func (s *liveStreamService) GetLiveStreamBySellerID(sellerID string) (*entities.LiveStream, error) {
	stream, err := s.repo.GetLiveStreamBySellerID(sellerID)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (s *liveStreamService) UpdateViewerCount(sellerID string, count int) error {
	err := s.repo.UpdateViewerCount(sellerID, count)
	if err != nil {
		return err
	}

	return nil
}