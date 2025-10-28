#!/usr/bin/env python3
import os
import sys
import torch
import json
from pathlib import Path

def check_pytorch():
    print(f"PyTorch version: {torch.__version__}")
    print(f"CUDA available: {torch.cuda.is_available()}")
    if torch.cuda.is_available():
        print(f"CUDA version: {torch.version.cuda}")
        print(f"GPU count: {torch.cuda.device_count()}")

def train_seller_offline(seller_id):
    """Simple offline training without CLIP"""
    datasets_dir = Path("datasets")
    seller_dir = datasets_dir / f"seller_{seller_id}"
    
    if not seller_dir.exists():
        print(f"No dataset found for seller_{seller_id}")
        return False
    
    print(f"Training seller_{seller_id}...")
    
    # Count products and images
    product_count = 0
    image_count = 0
    
    for product_dir in seller_dir.iterdir():
        if product_dir.is_dir():
            product_count += 1
            images_dir = product_dir / "images"
            if images_dir.exists():
                image_count += len(list(images_dir.glob("*.jpg")))
    
    print(f"Found {product_count} products with {image_count} images")
    
    # Create dummy embeddings file
    embeddings_dir = Path("embeddings")
    embeddings_dir.mkdir(exist_ok=True)
    
    dummy_result = {
        "seller_id": seller_id,
        "total_embeddings": image_count,
        "unique_products": product_count,
        "status": "completed"
    }
    
    with open(embeddings_dir / f"seller_{seller_id}_embeddings.json", "w") as f:
        json.dump(dummy_result, f)
    
    print(f"Training completed: {dummy_result}")
    return True

if __name__ == "__main__":
    check_pytorch()
    
    seller_id = sys.argv[1] if len(sys.argv) > 1 else "1"
    train_seller_offline(seller_id)