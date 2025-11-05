package entities

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
	SellerID        string `json:"seller_id"`
	TotalEmbeddings int    `json:"total_embeddings"`
	UniqueProducts  int    `json:"unique_products"`
	Status          string `json:"status"`
}