from ultralytics import YOLO
import numpy as np
from PIL import Image

class YOLODetector:
    def __init__(self, model_path="yolov8n.pt"):
        self.model = YOLO(model_path)
    
    def detect(self, image: Image.Image):
        """Detect objects in image and return bounding boxes"""
        results = self.model(np.array(image))
        
        detections = []
        for result in results:
            boxes = result.boxes
            if boxes is not None:
                for box in boxes:
                    x1, y1, x2, y2 = box.xyxy[0].cpu().numpy()
                    conf = box.conf[0].cpu().numpy()
                    detections.append([x1, y1, x2, y2, conf])
        
        return detections