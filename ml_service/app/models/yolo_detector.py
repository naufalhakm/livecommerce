from ultralytics import YOLO
import numpy as np
from PIL import Image

class YOLODetector:
    def __init__(self, model_path="yolov8n.pt"):
        self.model = YOLO(model_path)
    
    def detect(self, image: Image.Image):
        """Detect objects in image and return bounding boxes with class info"""
        results = self.model(np.array(image))
        
        detections = []
        for result in results:
            boxes = result.boxes
            if boxes is not None:
                for box in boxes:
                    x1, y1, x2, y2 = box.xyxy[0].cpu().numpy()
                    conf = box.conf[0].cpu().numpy()
                    cls = int(box.cls[0].cpu().numpy())
                    class_name = self.model.names[cls]
                    
                    detections.append({
                        'bbox': [int(x1), int(y1), int(x2), int(y2)],
                        'confidence': float(conf),
                        'class': class_name,
                        'class_id': cls
                    })
        
        return detections