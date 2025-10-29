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

def crop_objects_from_image(image_data, yolo_model):
    """Detect and crop objects from image using YOLO"""
    try:
        # Convert bytes to PIL Image
        image = Image.open(io.BytesIO(image_data))
        
        # Run YOLO detection
        results = yolo_model(image)
        
        cropped_objects = []
        for result in results:
            boxes = result.boxes
            if boxes is not None:
                for box in boxes:
                    # Get bounding box coordinates
                    x1, y1, x2, y2 = box.xyxy[0].cpu().numpy()
                    confidence = box.conf[0].cpu().numpy()
                    
                    # Only crop if confidence > 0.5
                    if confidence > 0.5:
                        # Crop the object
                        cropped = image.crop((int(x1), int(y1), int(x2), int(y2)))
                        
                        # Convert back to bytes
                        img_byte_arr = io.BytesIO()
                        cropped.save(img_byte_arr, format='JPEG')
                        cropped_objects.append(img_byte_arr.getvalue())
        
        return cropped_objects
    except Exception as e:
        print(f"Error cropping objects: {e}")
        return []

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
    
    # Initialize YOLO model
    yolo_model = init_yolo_model()
    if not yolo_model:
        print("Failed to load YOLO model, using original images")
    
    datasets_dir = Path("datasets")
    datasets_dir.mkdir(exist_ok=True)
    
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
            
            total_products += 1
    
    print(f"Dataset organized: {len(sellers)} sellers, {total_products} products, {total_images} images")
    return True

if __name__ == "__main__":
    check_pytorch()
    organize_dataset()