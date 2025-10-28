#!/usr/bin/env python3
import os
import sys
import torch
import json
import requests
from pathlib import Path
from urllib.parse import urlparse

def check_pytorch():
    print(f"PyTorch version: {torch.__version__}")
    print(f"CUDA available: {torch.cuda.is_available()}")
    if torch.cuda.is_available():
        print(f"CUDA version: {torch.version.cuda}")
        print(f"GPU count: {torch.cuda.device_count()}")

def fetch_products_from_api():
    """Fetch all products from backend API"""
    backend_url = os.getenv('BACKEND_URL', 'http://100.64.5.96:7080')
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
            
            # Download images
            if 'images' in product and product['images']:
                for i, image in enumerate(product['images']):
                    image_url = image['image_url']
                    filename = f"image_{i+1}.jpg"
                    filepath = images_dir / filename
                    
                    if download_image(image_url, filepath):
                        total_images += 1
                        print(f"Downloaded: {seller_id}/{product['name']}/{filename}")
            
            total_products += 1
    
    print(f"Dataset organized: {len(sellers)} sellers, {total_products} products, {total_images} images")
    return True

if __name__ == "__main__":
    check_pytorch()
    organize_dataset()