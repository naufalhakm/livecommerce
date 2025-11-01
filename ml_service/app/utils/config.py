import os
from pathlib import Path

class Config:
    def __init__(self):
        self.BASE_DIR = Path(__file__).parent.parent.parent
        self.DATASETS_DIR = self.BASE_DIR / "datasets"
        self.EMBEDDINGS_DIR = self.BASE_DIR / "embeddings"
        self.MODELS_DIR = self.BASE_DIR / "models"
        
        # Create directories if they don't exist
        self.DATASETS_DIR.mkdir(exist_ok=True)
        self.EMBEDDINGS_DIR.mkdir(exist_ok=True)
        self.MODELS_DIR.mkdir(exist_ok=True)
        
        # Model configurations
        self.YOLO_MODEL = "yolo11n.pt"
        self.CLIP_MODEL = "openai/clip-vit-base-patch32"
        self.EMBEDDING_DIM = 512
        
        # YOLO11 optimized settings
        self.YOLO_CONF_THRESHOLD = 0.4
        self.YOLO_IOU_THRESHOLD = 0.5
        self.MIN_OBJECT_SIZE = 30