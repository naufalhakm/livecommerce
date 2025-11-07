package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/services"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	metricsService *services.MetricsService
}

func NewExportHandler(metricsService *services.MetricsService) *ExportHandler {
	return &ExportHandler{
		metricsService: metricsService,
	}
}

func (h *ExportHandler) GenerateThesisResults(c *gin.Context) {
	// Create thesis_results directory
	resultsDir := "thesis_results"
	os.MkdirAll(resultsDir, 0755)

	// Get metrics data
	metricsData, err := h.metricsService.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}

	thesisData, err := h.metricsService.GetThesisData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get thesis data"})
		return
	}

	// Call ML service to generate charts and analysis
	mlServiceURL := os.Getenv("ML_SERVICE_URL")
	if mlServiceURL == "" {
		mlServiceURL = "http://localhost:8001"
	}
	
	resp, err := http.Post(mlServiceURL+"/generate-thesis-results", "application/json", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate charts from ML service"})
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	var mlResult map[string]interface{}
	json.Unmarshal(body, &mlResult)
	
	// Generate complete evaluation results JSON
	evaluationResults := map[string]interface{}{
		"generated_at": time.Now().Format("2006-01-02 15:04:05"),
		"metrics_summary": metricsData.Summary,
		"thesis_formatted": thesisData,
		"ml_generation_result": mlResult,
		"performance_analysis": map[string]interface{}{
			"yolo_avg_fps":           h.calculateAvgFPS(metricsData.YOLOMetrics),
			"clip_avg_similarity":    h.calculateAvgSimilarity(metricsData.CLIPMetrics),
			"system_avg_latency_ms":  metricsData.Summary.AvgSystemLatency,
			"total_frames_processed": metricsData.Summary.TotalFramesProcessed,
		},
	}

	// Save JSON file
	jsonFile := filepath.Join(resultsDir, "complete_evaluation_results.json")
	jsonData, _ := json.MarshalIndent(evaluationResults, "", "  ")
	os.WriteFile(jsonFile, jsonData, 0644)

	// Check which files actually exist in thesis_results directory
	generatedFiles := []string{}
	possibleFiles := []string{
		"yolo_performance.png",
		"clip_performance.png",
		"system_performance.png",
		"performance_comparison.png",
		"statistical_analysis.json",
		"complete_evaluation_results.json",
	}
	
	for _, filename := range possibleFiles {
		filePath := filepath.Join("thesis_results", filename)
		if _, err := os.Stat(filePath); err == nil {
			generatedFiles = append(generatedFiles, filename)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Thesis results generated successfully",
		"location": resultsDir,
		"files": generatedFiles,
		"evaluation_summary": evaluationResults,
	})
}

func (h *ExportHandler) DownloadThesisFile(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join("thesis_results", filename)
	
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	
	c.File(filePath)
}

func (h *ExportHandler) calculateAvgFPS(metrics []entities.YOLOMetric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	var total float64
	for _, m := range metrics {
		total += m.FPS
	}
	return total / float64(len(metrics))
}

func (h *ExportHandler) calculateAvgSimilarity(metrics []entities.CLIPMetric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	var total float64
	for _, m := range metrics {
		total += m.SimilarityScore
	}
	return total / float64(len(metrics))
}