import os
import json
import numpy as np
from PIL import Image
import io
from pathlib import Path
import logging
import requests

logger = logging.getLogger(__name__)

class TrainerService:
    def __init__(self, yolo_detector, clip_extractor, faiss_index, config):
        self.yolo_detector = yolo_detector
        self.clip_extractor = clip_extractor
        self.faiss_index = faiss_index
        self.config = config
    
    def fetch_products_from_api(self):
        """Fetch all products from backend API"""
        backend_url = os.getenv('BACKEND_URL', 'http://backend:8080')
        try:
            response = requests.get(f"{backend_url}/api/products")
            if response.status_code == 200:
                return response.json()
            else:
                logger.error(f"Failed to fetch products: {response.status_code}")
                return []
        except Exception as e:
            logger.error(f"Error fetching products: {e}")
            return []
    
    def crop_objects_from_image(self, image_data):
        """Detect and crop objects from image using YOLO"""
        try:
            image = Image.open(io.BytesIO(image_data))
            results = self.yolo_detector.model(image, conf=0.3)
            
            # Product-related classes (exclude person=0)
            product_classes = {
                24: 'handbag', 39: 'bottle', 40: 'wine glass', 41: 'cup', 42: 'fork', 43: 'knife', 44: 'spoon', 45: 'bowl',
                46: 'banana', 47: 'apple', 56: 'chair', 57: 'couch', 62: 'tv', 63: 'laptop', 64: 'mouse', 65: 'remote',
                66: 'keyboard', 67: 'cell phone', 73: 'book', 74: 'clock', 75: 'vase', 76: 'scissors', 77: 'teddy bear'
            }
            
            cropped_objects = []
            for result in results:
                boxes = result.boxes
                if boxes is not None:
                    for box in boxes:
                        class_id = int(box.cls[0].cpu().numpy())
                        confidence = box.conf[0].cpu().numpy()
                        
                        # Skip person class and low confidence
                        if class_id == 0 or confidence < 0.4:
                            continue
                        
                        x1, y1, x2, y2 = box.xyxy[0].cpu().numpy()
                        width = x2 - x1
                        height = y2 - y1
                        
                        if width < 50 or height < 50:
                            continue
                        
                        # Crop with padding
                        padding = 10
                        x1 = max(0, int(x1) - padding)
                        y1 = max(0, int(y1) - padding)
                        x2 = min(image.width, int(x2) + padding)
                        y2 = min(image.height, int(y2) + padding)
                        
                        cropped = image.crop((x1, y1, x2, y2))
                        
                        img_byte_arr = io.BytesIO()
                        cropped.save(img_byte_arr, format='JPEG')
                        cropped_objects.append(img_byte_arr.getvalue())
            
            return cropped_objects
        except Exception as e:
            logger.error(f"Error cropping objects: {e}")
            return []
    
    def organize_dataset_from_api(self):
        """Fetch products from API and organize into dataset structure"""
        logger.info("Fetching products from backend API...")
        products = self.fetch_products_from_api()
        
        if not products:
            logger.error("No products found from API")
            return False
        
        # Group products by seller
        sellers = {}
        for product in products:
            seller_id = f"seller_{product['seller_id']}"
            if seller_id not in sellers:
                sellers[seller_id] = []
            sellers[seller_id].append(product)
        
        total_products = 0
        total_images = 0
        
        # Create dataset structure
        for seller_id, seller_products in sellers.items():
            seller_dir = self.config.DATASETS_DIR / seller_id
            seller_dir.mkdir(exist_ok=True)
            
            for product in seller_products:
                product_dir = seller_dir / f"product_{product['id']}"
                images_dir = product_dir / "images"
                images_dir.mkdir(parents=True, exist_ok=True)
                
                # Create metadata.json
                metadata = {
                    "product_id": product['id'],
                    "product_name": product['name'],
                    "description": product['description'],
                    "price": product['price'],
                    "seller_id": product['seller_id']
                }
                
                with open(product_dir / "metadata.json", "w") as f:
                    json.dump(metadata, f, indent=2)
                
                # Download and process images
                if 'images' in product and product['images']:
                    for i, image in enumerate(product['images']):
                        image_url = image['image_url']
                        
                        try:
                            response = requests.get(image_url)
                            if response.status_code == 200:
                                image_data = response.content
                                
                                # Crop objects using YOLO
                                cropped_objects = self.crop_objects_from_image(image_data)
                                
                                if cropped_objects:
                                    # Save each cropped object
                                    for j, cropped_data in enumerate(cropped_objects):
                                        filename = f"image_{i+1}_crop_{j+1}.jpg"
                                        filepath = images_dir / filename
                                        
                                        with open(filepath, 'wb') as f:
                                            f.write(cropped_data)
                                        
                                        total_images += 1
                                        logger.info(f"Saved cropped: {seller_id}/{product['name']}/{filename}")
                                else:
                                    # Save original if no crops
                                    filename = f"image_{i+1}.jpg"
                                    filepath = images_dir / filename
                                    
                                    with open(filepath, 'wb') as f:
                                        f.write(image_data)
                                    
                                    total_images += 1
                                    logger.info(f"Saved original: {seller_id}/{product['name']}/{filename}")
                        
                        except Exception as e:
                            logger.error(f"Error processing image {image_url}: {e}")
                
                total_products += 1
        
        logger.info(f"Dataset organized: {len(sellers)} sellers, {total_products} products, {total_images} images")
        return True
    
    async def train_all_sellers(self):
        """Train models for all sellers from backend API"""
        # First organize dataset from API
        if not self.organize_dataset_from_api():
            raise ValueError("Failed to organize dataset from API")
        
        # Train each seller
        results = {}
        for seller_dir in self.config.DATASETS_DIR.iterdir():
            if seller_dir.is_dir() and seller_dir.name.startswith('seller_'):
                seller_id = seller_dir.name
                try:
                    result = await self.train_seller_model(seller_id)
                    results[seller_id] = result
                except Exception as e:
                    logger.error(f"Failed to train {seller_id}: {e}")
                    results[seller_id] = {"status": "error", "message": str(e)}
        
        return results
    
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