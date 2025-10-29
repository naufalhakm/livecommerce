#!/usr/bin/env python3
import os
import sys
import torch
import json
import requests
from pathlib import Path
from urllib.parse import urlparse
import cv2
import numpy as np
from ultralytics import YOLO
from PIL import Image
import io
from transformers import CLIPProcessor, CLIPModel
import numpy as np

def check_pytorch():
    print(f"PyTorch version: {torch.__version__}")
    print(f"CUDA available: {torch.cuda.is_available()}")
    if torch.cuda.is_available():
        print(f"CUDA version: {torch.version.cuda}")
        print(f"GPU count: {torch.cuda.device_count()}")

def init_yolo_model():
    """Initialize YOLO model for object detection"""
    try:
        model = YOLO('yolov8n.pt')
        print("YOLO model loaded successfully")
        return model
    except Exception as e:
        print(f"Error loading YOLO model: {e}")
        return None

def init_clip_model():
    """Initialize CLIP model for embedding generation"""
    try:
        model = CLIPModel.from_pretrained("openai/clip-vit-base-patch32")
        processor = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32")
        print("CLIP model loaded successfully")
        return model, processor
    except Exception as e:
        print(f"Error loading CLIP model: {e}")
        return None, None

def crop_objects_from_image(image_data, yolo_model):
    """Detect and crop objects from image using YOLO"""
    try:
        # Convert bytes to PIL Image
        image = Image.open(io.BytesIO(image_data))
        
        # Run YOLO detection with lower confidence for more detections
        results = yolo_model(image, conf=0.3)
        
        # Product-related classes (exclude person=0)
        product_classes = {
            39: 'bottle', 40: 'wine glass', 41: 'cup', 42: 'fork', 43: 'knife', 44: 'spoon', 45: 'bowl',
            46: 'banana', 47: 'apple', 48: 'sandwich', 49: 'orange', 50: 'broccoli', 51: 'carrot',
            52: 'hot dog', 53: 'pizza', 54: 'donut', 55: 'cake', 56: 'chair', 57: 'couch', 58: 'potted plant',
            59: 'bed', 60: 'dining table', 61: 'toilet', 62: 'tv', 63: 'laptop', 64: 'mouse', 65: 'remote',
            66: 'keyboard', 67: 'cell phone', 68: 'microwave', 69: 'oven', 70: 'toaster', 71: 'sink',
            72: 'refrigerator', 73: 'book', 74: 'clock', 75: 'vase', 76: 'scissors', 77: 'teddy bear',
            78: 'hair drier', 79: 'toothbrush'
        }
        
        cropped_objects = []
        for result in results:
            boxes = result.boxes
            if boxes is not None:
                for box in boxes:
                    # Get class ID and confidence
                    class_id = int(box.cls[0].cpu().numpy())
                    confidence = box.conf[0].cpu().numpy()
                    
                    # Skip person class (0) and low confidence
                    if class_id == 0 or confidence < 0.4:
                        continue
                    
                    # Get bounding box coordinates
                    x1, y1, x2, y2 = box.xyxy[0].cpu().numpy()
                    
                    # Check if bounding box is reasonable size
                    width = x2 - x1
                    height = y2 - y1
                    if width < 50 or height < 50:  # Skip very small objects
                        continue
                    
                    # Crop the object with some padding
                    padding = 10
                    x1 = max(0, int(x1) - padding)
                    y1 = max(0, int(y1) - padding)
                    x2 = min(image.width, int(x2) + padding)
                    y2 = min(image.height, int(y2) + padding)
                    
                    cropped = image.crop((x1, y1, x2, y2))
                    
                    # Convert back to bytes
                    img_byte_arr = io.BytesIO()
                    cropped.save(img_byte_arr, format='JPEG')
                    cropped_objects.append(img_byte_arr.getvalue())
                    
                    class_name = product_classes.get(class_id, f'class_{class_id}')
                    print(f"  Detected: {class_name} (conf: {confidence:.2f})")
        
        return cropped_objects
    except Exception as e:
        print(f"Error cropping objects: {e}")
        return []

def generate_image_embedding(image_data, clip_model, clip_processor):
    """Generate CLIP embedding from image"""
    try:
        image = Image.open(io.BytesIO(image_data))
        inputs = clip_processor(images=image, return_tensors="pt")
        
        with torch.no_grad():
            image_features = clip_model.get_image_features(**inputs)
            # Normalize the features
            image_features = image_features / image_features.norm(dim=-1, keepdim=True)
        
        return image_features.cpu().numpy()
    except Exception as e:
        print(f"Error generating image embedding: {e}")
        return None

def generate_text_embedding(text, clip_model, clip_processor):
    """Generate CLIP embedding from text"""
    try:
        text_inputs = clip_processor(text=[text], return_tensors="pt")
        
        with torch.no_grad():
            text_features = clip_model.get_text_features(**text_inputs)
            # Normalize the features
            text_features = text_features / text_features.norm(dim=-1, keepdim=True)
        
        return text_features.cpu().numpy()
    except Exception as e:
        print(f"Error generating text embedding: {e}")
        return None

def fetch_products_from_api():
    """Fetch all products from backend API"""
    backend_url = os.getenv('BACKEND_URL', 'http://100.64.5.96:7080')
    print(f"Using backend URL: {backend_url}")
    try:
        response = requests.get(f"{backend_url}/api/products")
        if response.status_code == 200:
            return response.json()
        else:
            print(f"Failed to fetch products: {response.status_code}")
            return []
    except Exception as e:
        print(f"Error fetching products: {e}")
        return []

def download_image(url, filepath):
    """Download image from URL"""
    try:
        response = requests.get(url)
        if response.status_code == 200:
            with open(filepath, 'wb') as f:
                f.write(response.content)
            return True
    except Exception as e:
        print(f"Error downloading {url}: {e}")
    return False

def organize_dataset():
    """Fetch products from API and organize into dataset structure"""
    print("Fetching products from backend API...")
    products = fetch_products_from_api()
    
    if not products:
        print("No products found from API")
        return False
    
    # Initialize models
    yolo_model = init_yolo_model()
    if not yolo_model:
        print("Failed to load YOLO model, using original images")
    
    clip_model, clip_processor = init_clip_model()
    if not clip_model:
        print("Failed to load CLIP model, skipping embedding generation")
    
    datasets_dir = Path("datasets")
    embeddings_dir = Path("embeddings")
    datasets_dir.mkdir(exist_ok=True)
    embeddings_dir.mkdir(exist_ok=True)
    
    total_products = 0
    total_images = 0
    
    # Group products by seller
    sellers = {}
    for product in products:
        seller_id = f"seller_{product['seller_id']}"
        if seller_id not in sellers:
            sellers[seller_id] = []
        sellers[seller_id].append(product)
    
    # Create dataset structure
    for seller_id, seller_products in sellers.items():
        seller_dir = datasets_dir / seller_id
        seller_dir.mkdir(exist_ok=True)
        
        for product in seller_products:
            product_dir = seller_dir / f"product_{product['id']}"
            images_dir = product_dir / "images"
            model_dir = product_dir / "model"
            
            # Create embedding directory for this seller
            seller_embedding_dir = embeddings_dir / seller_id
            seller_embedding_dir.mkdir(exist_ok=True)
            
            images_dir.mkdir(parents=True, exist_ok=True)
            model_dir.mkdir(parents=True, exist_ok=True)
            
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
                        # Download image data
                        response = requests.get(image_url)
                        if response.status_code == 200:
                            image_data = response.content
                            
                            # Crop objects using YOLO if model is available
                            if yolo_model:
                                cropped_objects = crop_objects_from_image(image_data, yolo_model)
                                
                                # Save each cropped object
                                for j, cropped_data in enumerate(cropped_objects):
                                    filename = f"image_{i+1}_crop_{j+1}.jpg"
                                    filepath = images_dir / filename
                                    
                                    with open(filepath, 'wb') as f:
                                        f.write(cropped_data)
                                    
                                    total_images += 1
                                    print(f"Saved cropped: {seller_id}/{product['name']}/{filename}")
                            else:
                                # Save original image if YOLO not available
                                filename = f"image_{i+1}.jpg"
                                filepath = images_dir / filename
                                
                                with open(filepath, 'wb') as f:
                                    f.write(image_data)
                                
                                total_images += 1
                                print(f"Saved original: {seller_id}/{product['name']}/{filename}")
                    
                    except Exception as e:
                        print(f"Error processing image {image_url}: {e}")
            
            # Generate text embedding for product
            if clip_model and clip_processor:
                product_text = f"{product['name']} {product['description']}"
                text_embedding = generate_text_embedding(product_text, clip_model, clip_processor)
                
                if text_embedding is not None:
                    text_embedding_path = seller_embedding_dir / f"product_{product['id']}_text.npy"
                    np.save(text_embedding_path, text_embedding)
                    print(f"Saved text embedding: {text_embedding_path}")
                
                # Generate image embeddings from saved cropped images
                for image_file in images_dir.glob("*.jpg"):
                    try:
                        with open(image_file, 'rb') as f:
                            image_data = f.read()
                        
                        image_embedding = generate_image_embedding(image_data, clip_model, clip_processor)
                        
                        if image_embedding is not None:
                            embedding_filename = f"product_{product['id']}_{image_file.stem}_image.npy"
                            image_embedding_path = seller_embedding_dir / embedding_filename
                            np.save(image_embedding_path, image_embedding)
                            print(f"Saved image embedding: {embedding_filename}")
                    
                    except Exception as e:
                        print(f"Error generating embedding for {image_file}: {e}")
            
            total_products += 1
    
    print(f"Dataset organized: {len(sellers)} sellers, {total_products} products, {total_images} images")
    return True

if __name__ == "__main__":
    check_pytorch()
    organize_dataset()