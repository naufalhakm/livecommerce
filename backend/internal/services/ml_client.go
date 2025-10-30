package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type MLClient struct {
	BaseURL string
}

type PredictionResponse struct {
	Predictions []struct {
		BBox            []int   `json:"bbox"`
		ProductID       string  `json:"product_id"`
		ProductName     string  `json:"product_name"`
		Price           float64 `json:"price"`
		Confidence      float64 `json:"confidence"`
		SimilarityScore float64 `json:"similarity_score"`
	} `json:"predictions"`
	Detections []struct {
		BBox       []int   `json:"bbox"`
		Confidence float64 `json:"confidence"`
		Class      string  `json:"class"`
		ClassID    int     `json:"class_id"`
	} `json:"detections"`
	TotalDetections int `json:"total_detections"`
	TotalProducts   int `json:"total_products"`
}

type TrainingResponse struct {
	SellerID         string `json:"seller_id"`
	TotalEmbeddings  int    `json:"total_embeddings"`
	UniqueProducts   int    `json:"unique_products"`
	Status           string `json:"status"`
}

func NewMLClient() *MLClient {
	baseURL := os.Getenv("ML_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://ml_service:8001"
	}
	return &MLClient{BaseURL: baseURL}
}

func (c *MLClient) TrainModel(sellerID string) (*TrainingResponse, error) {
	url := fmt.Sprintf("%s/train?seller_id=%s", c.BaseURL, sellerID)
	
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ML service returned status: %d", resp.StatusCode)
	}

	var result TrainingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *MLClient) PredictProduct(sellerID string, imageData []byte) (*PredictionResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add seller_id field
	writer.WriteField("seller_id", sellerID)

	// Add file field
	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		return nil, err
	}
	part.Write(imageData)
	writer.Close()

	url := fmt.Sprintf("%s/predict", c.BaseURL)
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

	var result PredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *MLClient) GetTrainingStatus(sellerID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/training-status/%s", c.BaseURL, sellerID)
	
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

func (c *MLClient) ProcessStreamFrame(sellerID string, imageData []byte) (*PredictionResponse, error) {
	// Process live stream frame for object detection and product recognition
	return c.PredictProduct(sellerID, imageData)
}