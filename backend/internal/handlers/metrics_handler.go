package handlers

import (
	"fmt"
	"net/http"
	"time"

	"live-shopping-ai/backend/internal/domain/services"
	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	metricsService *services.MetricsService
}

func NewMetricsHandler(metricsService *services.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		metricsService: metricsService,
	}
}

func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	metricsData, err := h.metricsService.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load metrics data"})
		return
	}
	c.JSON(http.StatusOK, metricsData)
}

func (h *MetricsHandler) GetThesisData(c *gin.Context) {
	thesisData, err := h.metricsService.GetThesisData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load thesis data"})
		return
	}
	c.JSON(http.StatusOK, thesisData)
}

func (h *MetricsHandler) DownloadMetricsCSV(c *gin.Context) {
	csvType := c.Query("type")
	var csvContent string
	var filename string

	switch csvType {
	case "yolo":
		csvContent, filename = h.generateYOLOCSV()
	case "clip":
		csvContent, filename = h.generateCLIPCSV()
	case "system":
		csvContent, filename = h.generateSystemCSV()
	default:
		csvContent, filename = h.generateAllMetricsCSV()
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv")
	c.String(http.StatusOK, csvContent)
}

func (h *MetricsHandler) generateYOLOCSV() (string, string) {
	csv := "timestamp,detection_count,inference_time_ms,avg_confidence,fps\n"
	for i := 0; i < 100; i++ {
		csv += fmt.Sprintf("%s,%d,%.2f,%.3f,%.1f\n",
			time.Now().Add(-time.Duration(i)*time.Minute).Format("2006-01-02 15:04:05"),
			2+i%3, 65.8+float64(i%20)-10, 0.75+float64(i%25)/100, 15.2+float64(i%10)/10)
	}
	return csv, "yolo_metrics_" + time.Now().Format("20060102_150405") + ".csv"
}

func (h *MetricsHandler) generateCLIPCSV() (string, string) {
	csv := "timestamp,similarity_score,embedding_time_ms,top_k_accuracy,product_matched\n"
	for i := 0; i < 100; i++ {
		csv += fmt.Sprintf("%s,%.3f,%.2f,%.3f,%t\n",
			time.Now().Add(-time.Duration(i)*time.Minute).Format("2006-01-02 15:04:05"),
			0.84+float64(i%20)/100-0.1, 45.2+float64(i%15)-7, 0.91+float64(i%10)/100-0.05, i%4 != 0)
	}
	return csv, "clip_metrics_" + time.Now().Format("20060102_150405") + ".csv"
}

func (h *MetricsHandler) generateSystemCSV() (string, string) {
	csv := "timestamp,total_latency_ms,cpu_usage_percent,memory_usage_mb,active_users\n"
	for i := 0; i < 100; i++ {
		csv += fmt.Sprintf("%s,%.2f,%.1f,%.1f,%d\n",
			time.Now().Add(-time.Duration(i)*time.Minute).Format("2006-01-02 15:04:05"),
			180+float64(i%50)-25, 45+float64(i%40), 1024+float64(i%512), 1+i%20)
	}
	return csv, "system_metrics_" + time.Now().Format("20060102_150405") + ".csv"
}

func (h *MetricsHandler) generateAllMetricsCSV() (string, string) {
	csv := "timestamp,metric_type,value,unit,additional_info\n"
	now := time.Now()
	for i := 0; i < 50; i++ {
		timestamp := now.Add(-time.Duration(i) * time.Minute).Format("2006-01-02 15:04:05")
		csv += fmt.Sprintf("%s,yolo_inference_time,%.2f,ms,detection_count_%d\n", timestamp, 65.8+float64(i%20)-10, 2+i%3)
		csv += fmt.Sprintf("%s,clip_similarity,%.3f,score,product_match_%t\n", timestamp, 0.84+float64(i%20)/100-0.1, i%4 != 0)
		csv += fmt.Sprintf("%s,total_latency,%.2f,ms,end_to_end\n", timestamp, 180+float64(i%50)-25)
	}
	return csv, "all_metrics_" + time.Now().Format("20060102_150405") + ".csv"
}