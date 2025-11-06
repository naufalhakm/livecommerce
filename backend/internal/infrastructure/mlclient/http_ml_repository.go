package mlclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/repositories"
	"mime/multipart"
	"net/http"
	"os"
)

type httpMLRepository struct {
	baseURL string
}

func NewHttpMLRepository() repositories.MLRepository {
	baseURL := os.Getenv("ML_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://ml_service:8001"
	}
	return &httpMLRepository{baseURL: baseURL}
}

func (r *httpMLRepository) TrainModel(sellerID string) (*entities.TrainingResponse, error) {
	url := fmt.Sprintf("%s/train?seller_id=%s", r.baseURL, sellerID)
	
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned status: %d", resp.StatusCode)
	}

	var result entities.TrainingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *httpMLRepository) PredictProduct(sellerID string, imageData []byte) (*entities.PredictionResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("seller_id", sellerID)
	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		return nil, err
	}
	part.Write(imageData)
	writer.Close()

	url := fmt.Sprintf("%s/predict", r.baseURL)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML service returned status: %d, body: %s", resp.StatusCode, string(body))
	}

	var result entities.PredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *httpMLRepository) GetTrainingStatus(sellerID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/training-status/%s", r.baseURL, sellerID)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *httpMLRepository) ProcessStreamFrame(sellerID string, imageData []byte) (*entities.PredictionResponse, error) {
	return r.PredictProduct(sellerID, imageData)
}