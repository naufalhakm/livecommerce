import json
import os
import time
from datetime import datetime
from typing import Dict, List, Any
import psutil

class MetricsLogger:
    def __init__(self):
        self.logs_dir = "experiment_logs"
        os.makedirs(self.logs_dir, exist_ok=True)
        
    def log_yolo_metrics(self, detections: List, inference_time: float):
        """Log YOLO detection metrics"""
        metric = {
            "timestamp": datetime.now().isoformat(),
            "detection_count": len(detections),
            "inference_time_ms": inference_time * 1000,
            "avg_confidence": sum([d.get('confidence', 0) for d in detections]) / max(len(detections), 1),
            "fps": 1.0 / max(inference_time, 0.001)
        }
        self._append_to_file("yolo_metrics.json", metric)
        
    def log_clip_metrics(self, similarity_scores: List[float], embedding_time: float, 
                        top_k_results: List, product_matched: bool):
        """Log CLIP recognition metrics"""
        metric = {
            "timestamp": datetime.now().isoformat(),
            "similarity_score": max(similarity_scores) if similarity_scores else 0,
            "embedding_time_ms": embedding_time * 1000,
            "top_k_accuracy": len([s for s in similarity_scores if s > 0.8]) / max(len(similarity_scores), 1),
            "product_matched": product_matched
        }
        self._append_to_file("clip_metrics.json", metric)
        
    def log_system_performance(self, frame_time: float, total_latency: float, 
                             fps: float, memory_mb: float, cpu_percent: float):
        """Log system performance metrics"""
        metric = {
            "timestamp": datetime.now().isoformat(),
            "total_latency_ms": total_latency * 1000,
            "cpu_usage_percent": cpu_percent,
            "memory_usage_mb": memory_mb,
            "active_users": 1,  # Will be updated by backend
            "fps": fps
        }
        self._append_to_file("system_metrics.json", metric)
        
    def _append_to_file(self, filename: str, data: Dict):
        """Append data to JSON file"""
        filepath = os.path.join(self.logs_dir, filename)
        
        # Read existing data
        if os.path.exists(filepath):
            with open(filepath, 'r') as f:
                try:
                    existing_data = json.load(f)
                    if not isinstance(existing_data, list):
                        existing_data = []
                except:
                    existing_data = []
        else:
            existing_data = []
            
        # Append new data
        existing_data.append(data)
        
        # Keep only last 1000 entries
        if len(existing_data) > 1000:
            existing_data = existing_data[-1000:]
            
        # Write back
        with open(filepath, 'w') as f:
            json.dump(existing_data, f, indent=2)

# Global instance
metrics_logger = MetricsLogger()