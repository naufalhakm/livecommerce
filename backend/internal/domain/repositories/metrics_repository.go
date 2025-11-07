package repositories

import "live-shopping-ai/backend/internal/domain/entities"

type MetricsRepository interface {
	GetMetricsData() (*entities.MetricsData, error)
	GetThesisFormattedData(*entities.MetricsData) map[string]interface{}
}