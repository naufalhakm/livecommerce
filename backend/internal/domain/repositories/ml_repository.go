package repositories

import (
	"live-shopping-ai/backend/internal/domain/entities"
	"mime/multipart"
)

type MLRepository interface {
	TrainModel(sellerID string) (*entities.TrainingResponse, error)
	PredictProduct(sellerID string, imageData []byte) (*entities.PredictionResponse, error)
	GetTrainingStatus(sellerID string) (map[string]interface{}, error)
	ProcessStreamFrame(sellerID string, imageData []byte) (*entities.PredictionResponse, error)
}

type StorageRepository interface {
	UploadFromForm(file *multipart.FileHeader) (string, error)
}