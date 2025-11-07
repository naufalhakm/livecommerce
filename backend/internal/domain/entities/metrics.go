package entities

import "time"

type YOLOMetric struct {
	Timestamp      time.Time `json:"timestamp"`
	DetectionCount int       `json:"detection_count"`
	InferenceTime  float64   `json:"inference_time_ms"`
	Confidence     float64   `json:"avg_confidence"`
	FPS            float64   `json:"fps"`
}

type CLIPMetric struct {
	Timestamp       time.Time `json:"timestamp"`
	SimilarityScore float64   `json:"similarity_score"`
	EmbeddingTime   float64   `json:"embedding_time_ms"`
	TopKAccuracy    float64   `json:"top_k_accuracy"`
	ProductMatched  bool      `json:"product_matched"`
}

type SystemMetric struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalLatency float64   `json:"total_latency_ms"`
	CPUUsage     float64   `json:"cpu_usage_percent"`
	MemoryUsage  float64   `json:"memory_usage_mb"`
	ActiveUsers  int       `json:"active_users"`
}

type MetricsData struct {
	YOLOMetrics   []YOLOMetric   `json:"yolo_metrics"`
	CLIPMetrics   []CLIPMetric   `json:"clip_metrics"`
	SystemMetrics []SystemMetric `json:"system_metrics"`
	Summary       MetricsSummary `json:"summary"`
}

type MetricsSummary struct {
	TotalFramesProcessed int     `json:"total_frames_processed"`
	AvgYOLOLatency      float64 `json:"avg_yolo_latency_ms"`
	AvgCLIPLatency      float64 `json:"avg_clip_latency_ms"`
	AvgSystemLatency    float64 `json:"avg_system_latency_ms"`
	YOLOPrecision       float64 `json:"yolo_precision"`
	YOLORecall          float64 `json:"yolo_recall"`
	YOLOF1Score         float64 `json:"yolo_f1_score"`
	CLIPTop1Accuracy    float64 `json:"clip_top1_accuracy"`
	CLIPTop3Accuracy    float64 `json:"clip_top3_accuracy"`
	CLIPTop5Accuracy    float64 `json:"clip_top5_accuracy"`
	SystemUptime        string  `json:"system_uptime"`
	LastUpdated         time.Time `json:"last_updated"`
}