package services

import (
	"context"
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/repositories"
	"mime/multipart"
)

type StreamService interface {
	ProcessStreamFrame(ctx context.Context, sellerID string, frame *multipart.FileHeader) (*entities.PredictionResponse, error)
	PredictFrame(ctx context.Context, sellerID string, frame *multipart.FileHeader) (*entities.PredictionResponse, error)
}

type streamService struct {
	mlRepo    repositories.MLRepository
	pinnedRepo repositories.PinnedProductRepository
}

func NewStreamService(mlRepo repositories.MLRepository, pinnedRepo repositories.PinnedProductRepository) StreamService {
	return &streamService{
		mlRepo:    mlRepo,
		pinnedRepo: pinnedRepo,
	}
}

func (s *streamService) ProcessStreamFrame(ctx context.Context, sellerID string, frame *multipart.FileHeader) (*entities.PredictionResponse, error) {
	src, err := frame.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	frameData := make([]byte, frame.Size)
	if _, err := src.Read(frameData); err != nil {
		return nil, err
	}

	return s.mlRepo.ProcessStreamFrame(sellerID, frameData)
}

func (s *streamService) PredictFrame(ctx context.Context, sellerID string, frame *multipart.FileHeader) (*entities.PredictionResponse, error) {
	src, err := frame.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	frameData := make([]byte, frame.Size)
	if _, err := src.Read(frameData); err != nil {
		return nil, err
	}

	return s.mlRepo.PredictProduct(sellerID, frameData)
}