import os
import json
import numpy as np
from PIL import Image
import io
from pathlib import Path
import logging

logger = logging.getLogger(__name__)

class TrainerService:
    def __init__(self, yolo_detector, clip_extractor, faiss_index, config):
        self.yolo_detector = yolo_detector
        self.clip_extractor = clip_extractor
        self.faiss_index = faiss_index
        self.config = config
    
    async def train_seller_model(self, seller_id: str):
        """Train model for seller using product_id structure"""
        from main import training_status
        
        seller_path = self.config.DATASETS_DIR / seller_id
        if not seller_path.exists():
            raise ValueError(f"Seller dataset not found: {seller_id}")
        
        training_status[seller_id] = {"status": "training", "progress": 20, "message": "Processing images..."}
        
        embeddings = []
        product_metadata = []
        
        # Process each product folder
        for product_dir in seller_path.iterdir():
            if not product_dir.is_dir():
                continue
            
            product_id = product_dir.name
            metadata_file = product_dir / "metadata.json"
            images_dir = product_dir / "images"
            
            # Load product metadata
            if metadata_file.exists():
                with open(metadata_file, 'r') as f:
                    metadata = json.load(f)
                product_name = metadata.get("product_name", product_id)
            else:
                product_name = product_id
            
            # Process images
            if images_dir.exists():
                for img_file in images_dir.glob("*.jpg"):
                    try:
                        image = Image.open(img_file).convert('RGB')
                        embedding = self.clip_extractor.extract_embedding(image)
                        embeddings.append(embedding)
                        product_metadata.append({
                            "product_id": product_id,
                            "product_name": product_name
                        })
                    except Exception as e:
                        logger.error(f"Error processing {img_file}: {e}")
        
        if not embeddings:
            raise ValueError("No valid images found for training")
        
        # Save FAISS index
        from main import training_status
        training_status[seller_id] = {"status": "training", "progress": 80, "message": "Building FAISS index..."}
        
        embeddings_array = np.array(embeddings)
        self.faiss_index.save_seller_index(seller_id, embeddings_array, product_metadata)
        
        return {
            "seller_id": seller_id,
            "total_embeddings": len(embeddings),
            "unique_products": len(set(p["product_id"] for p in product_metadata)),
            "status": "success"
        }
    
    async def detect_products(self, seller_id: str, image_data: bytes):
        """Detect products in image"""
        if seller_id not in self.faiss_index.seller_indices:
            raise ValueError(f"No trained model found for seller: {seller_id}")
        
        # Convert bytes to PIL Image
        image = Image.open(io.BytesIO(image_data)).convert('RGB')
        
        # YOLO detection
        detections = self.yolo_detector.detect(image)
        
        predictions = []
        for detection in detections:
            x1, y1, x2, y2, conf = detection
            
            # Extract detected region
            detected_region = image.crop((int(x1), int(y1), int(x2), int(y2)))
            
            # Get embedding and search
            embedding = self.clip_extractor.extract_embedding(detected_region)
            results = self.faiss_index.search(seller_id, embedding, k=1)
            
            if results:
                result = results[0]
                predictions.append({
                    "bbox": [int(x1), int(y1), int(x2), int(y2)],
                    "product_id": result["product_id"],
                    "product_name": result["product_name"],
                    "confidence": float(conf),
                    "similarity_score": result["similarity_score"]
                })
        
        return {"predictions": predictions}