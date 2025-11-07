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
        """Detect and crop objects from image using YOLO11"""
        try:
            image = Image.open(io.BytesIO(image_data))
            
            # Use optimized YOLO11 detection
            detections = self.yolo_detector.detect(
                image, 
                conf_threshold=self.config.YOLO_CONF_THRESHOLD,
                iou_threshold=self.config.YOLO_IOU_THRESHOLD
            )
            
            # Enhanced product classes for e-commerce
            product_classes = {
                # Electronics
                62: 'tv', 63: 'laptop', 64: 'mouse', 65: 'remote', 66: 'keyboard', 67: 'cell phone',
                # Fashion & Accessories
                24: 'handbag', 25: 'tie', 26: 'suitcase',
                # Kitchen & Dining
                39: 'bottle', 40: 'wine glass', 41: 'cup', 42: 'fork', 43: 'knife', 44: 'spoon', 45: 'bowl',
                68: 'microwave', 69: 'oven', 70: 'toaster', 72: 'refrigerator',
                # Home & Garden
                56: 'chair', 57: 'couch', 58: 'potted plant', 59: 'bed', 60: 'dining table',
                73: 'book', 74: 'clock', 75: 'vase', 76: 'scissors', 77: 'teddy bear',
                # Food Items
                46: 'banana', 47: 'apple', 48: 'sandwich', 49: 'orange', 50: 'broccoli', 51: 'carrot',
                52: 'hot dog', 53: 'pizza', 54: 'donut', 55: 'cake'
            }
            
            cropped_objects = []
            for detection in detections:
                class_id = detection['class_id']
                confidence = detection['confidence']
                
                # Skip person class and focus on products
                if class_id == 0 or class_id not in product_classes:
                    continue
                
                x1, y1, x2, y2 = detection['bbox']
                width, height = x2 - x1, y2 - y1
                
                # Enhanced size filtering
                if width < self.config.MIN_OBJECT_SIZE or height < self.config.MIN_OBJECT_SIZE:
                    continue
                
                # Smart padding based on object size
                padding = max(5, min(20, int(min(width, height) * 0.1)))
                x1 = max(0, x1 - padding)
                y1 = max(0, y1 - padding)
                x2 = min(image.width, x2 + padding)
                y2 = min(image.height, y2 + padding)
                
                cropped = image.crop((x1, y1, x2, y2))
                
                # Resize if too large (optimize for CLIP)
                if cropped.width > 512 or cropped.height > 512:
                    cropped.thumbnail((512, 512), Image.Resampling.LANCZOS)
                
                img_byte_arr = io.BytesIO()
                cropped.save(img_byte_arr, format='JPEG', quality=85, optimize=True)
                cropped_objects.append(img_byte_arr.getvalue())
                
                logger.info(f"Cropped {product_classes[class_id]} (conf: {confidence:.2f})")
            
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
                                
                                # Always save original image for training
                                filename = f"image_{i+1}.jpg"
                                filepath = images_dir / filename
                                
                                with open(filepath, 'wb') as f:
                                    f.write(image_data)
                                
                                total_images += 1
                                logger.info(f"Saved: {seller_id}/{product['name']}/{filename}")
                                
                                # Also try to crop objects using YOLO for additional training data
                                cropped_objects = self.crop_objects_from_image(image_data)
                                for j, cropped_data in enumerate(cropped_objects):
                                    crop_filename = f"image_{i+1}_crop_{j+1}.jpg"
                                    crop_filepath = images_dir / crop_filename
                                    
                                    with open(crop_filepath, 'wb') as f:
                                        f.write(cropped_data)
                                    
                                    total_images += 1
                                    logger.info(f"Saved crop: {seller_id}/{product['name']}/{crop_filename}")
                        
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
        seller_path = self.config.DATASETS_DIR / seller_id
        if not seller_path.exists():
            raise ValueError(f"Seller dataset not found: {seller_id}")
        
        embeddings = []
        product_metadata = []
        
        logger.info(f"Training {seller_id}...")
        
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
                price = metadata.get("price", 0.0)
            else:
                product_name = product_id
                price = 0.0
            
            # Process images
            if images_dir.exists():
                image_count = 0
                for img_file in images_dir.glob("*.jpg"):
                    try:
                        image = Image.open(img_file).convert('RGB')
                        embedding = self.clip_extractor.extract_embedding(image)
                        embeddings.append(embedding)
                        product_metadata.append({
                            "product_id": product_id,
                            "product_name": product_name,
                            "price": price
                        })
                        image_count += 1
                        logger.info(f"Processed {img_file.name} for {product_name}")
                    except Exception as e:
                        logger.error(f"Error processing {img_file}: {e}")
                
                logger.info(f"Product {product_name}: {image_count} images processed")
        
        if not embeddings:
            raise ValueError(f"No valid images found for training {seller_id}")
        
        logger.info(f"Building FAISS index for {seller_id} with {len(embeddings)} embeddings...")
        
        embeddings_array = np.array(embeddings)
        self.faiss_index.save_seller_index(seller_id, embeddings_array, product_metadata)
        
        logger.info(f"Training completed for {seller_id}")
        
        return {
            "seller_id": seller_id,
            "total_embeddings": len(embeddings),
            "unique_products": len(set(p["product_id"] for p in product_metadata)),
            "status": "success"
        }
    
    async def detect_products(self, seller_id: str, image_data: bytes):
        """Detect products in image with timing metrics"""
        import time
        
        logger.info(f"🔍 DETECT_PRODUCTS: seller_id={seller_id}")
        logger.info(f"📊 Available indices: {list(self.faiss_index.seller_indices.keys())}")
        
        if seller_id not in self.faiss_index.seller_indices:
            logger.error(f"❌ No trained model found for seller: {seller_id}")
            raise ValueError(f"No trained model found for seller: {seller_id}")
        
        logger.info(f"✅ Found model for {seller_id}")
        
        # Convert bytes to PIL Image
        logger.info(f"📷 Converting image data to PIL Image")
        image = Image.open(io.BytesIO(image_data)).convert('RGB')
        logger.info(f"📷 Image size: {image.size}")
        
        # YOLO11 optimized detection with timing
        logger.info(f"🤖 Running YOLO11 detection")
        yolo_start = time.time()
        detections = self.yolo_detector.detect(
            image,
            conf_threshold=self.config.YOLO_CONF_THRESHOLD,
            iou_threshold=self.config.YOLO_IOU_THRESHOLD
        )
        yolo_time = time.time() - yolo_start
        logger.info(f"🎯 YOLO11 found {len(detections)} detections in {yolo_time:.3f}s")
        
        predictions = []
        all_detections = []
        clip_total_time = 0
        
        # Process YOLO detections (now returns structured data)
        for detection in detections:
            # Skip person class for product detection
            if detection.get('class_id') == 0:
                continue
                
            bbox = detection['bbox']
            x1, y1, x2, y2 = bbox
            
            # Add to all detections for object tracking
            all_detections.append(detection)
            
            # Extract detected region for product recognition
            detected_region = image.crop((x1, y1, x2, y2))
            
            # Get embedding and search with timing
            logger.info(f"🧪 Extracting CLIP embedding for detection {len(predictions)+1}")
            clip_start = time.time()
            embedding = self.clip_extractor.extract_embedding(detected_region)
            logger.info(f"🔍 Searching FAISS index for {seller_id}")
            results = self.faiss_index.search(seller_id, embedding, k=1)
            clip_time = time.time() - clip_start
            clip_total_time += clip_time
            logger.info(f"📊 FAISS results: {results} (took {clip_time:.3f}s)")
            
            if results and results[0]["similarity_score"] > 0.7:  # Threshold for product match
                result = results[0]
                # Extract numeric product ID from product_X format
                product_id_str = result["product_id"].replace("product_", "")
                
                predictions.append({
                    "bbox": bbox,
                    "product_id": product_id_str,
                    "product_name": result["product_name"],
                    "price": result.get("price", 0.0),
                    "confidence": detection['confidence'],
                    "similarity_score": result["similarity_score"]
                })
        
        logger.info(f"✅ Final predictions: {len(predictions)} items, detections: {len(all_detections)}")
        return {
            "predictions": predictions,
            "detections": all_detections,
            "yolo_time": yolo_time,
            "clip_time": clip_total_time / max(len(all_detections), 1)  # Average CLIP time per detection
        }