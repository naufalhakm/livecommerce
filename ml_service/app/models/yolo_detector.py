from ultralytics import YOLO
import numpy as np
from PIL import Image

class YOLODetector:
    def __init__(self, model_path="yolo11n.pt"):
        self.model = YOLO(model_path)
        # Optimize for inference
        self.model.fuse()
    
    def detect(self, image: Image.Image, conf_threshold=0.4, iou_threshold=0.5):
        """Detect objects in image with optimized thresholds"""
        # YOLO11 optimized inference
        results = self.model(
            np.array(image),
            conf=conf_threshold,
            iou=iou_threshold,
            verbose=False,
            device='cpu'  # Explicit CPU for consistency
        )
        
        detections = []
        for result in results:
            boxes = result.boxes
            if boxes is not None:
                for box in boxes:
                    x1, y1, x2, y2 = box.xyxy[0].cpu().numpy()
                    conf = box.conf[0].cpu().numpy()
                    cls = int(box.cls[0].cpu().numpy())
                    class_name = self.model.names[cls]
                    
                    # Filter small objects and low confidence
                    width, height = x2 - x1, y2 - y1
                    if width > 30 and height > 30 and conf > conf_threshold:
                        detections.append({
                            'bbox': [int(x1), int(y1), int(x2), int(y2)],
                            'confidence': float(conf),
                            'class': class_name,
                            'class_id': cls,
                            'area': int(width * height)
                        })
        
        # Sort by confidence (highest first)
        detections.sort(key=lambda x: x['confidence'], reverse=True)
        return detections