import torch
import torch.nn as nn
from torch.utils.data import Dataset, DataLoader
from PIL import Image
import json
import logging
from pathlib import Path
import numpy as np
from models.fine_tuned_clip import FineTunedCLIP

logger = logging.getLogger(__name__)

class ProductDataset(Dataset):
    def __init__(self, dataset_dir, processor):
        self.dataset_dir = Path(dataset_dir)
        self.processor = processor
        self.samples = []
        self.product_to_id = {}
        
        # Build dataset
        product_id = 0
        for product_dir in self.dataset_dir.glob("product_*"):
            if not product_dir.is_dir():
                continue
                
            # Load metadata
            metadata_file = product_dir / "metadata.json"
            if metadata_file.exists():
                with open(metadata_file) as f:
                    metadata = json.load(f)
                product_name = metadata.get("product_name", product_dir.name)
            else:
                product_name = product_dir.name
            
            self.product_to_id[product_dir.name] = product_id
            
            # Add images
            images_dir = product_dir / "images"
            if images_dir.exists():
                for img_file in images_dir.glob("*.jpg"):
                    self.samples.append({
                        'image_path': img_file,
                        'product_id': product_id,
                        'product_name': product_name
                    })
            
            product_id += 1
    
    def __len__(self):
        return len(self.samples)
    
    def __getitem__(self, idx):
        sample = self.samples[idx]
        
        # Load and process image
        image = Image.open(sample['image_path']).convert('RGB')
        inputs = self.processor(images=image, return_tensors="pt")
        
        return {
            'pixel_values': inputs['pixel_values'].squeeze(0),
            'product_id': torch.tensor(sample['product_id'], dtype=torch.long),
            'product_name': sample['product_name']
        }

class FineTuner:
    def __init__(self, config):
        self.config = config
        self.device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    
    def fine_tune_seller_model(self, seller_id: str, epochs=10, learning_rate=1e-4):
        """Fine-tune CLIP model for specific seller"""
        seller_path = self.config.DATASETS_DIR / seller_id
        if not seller_path.exists():
            raise ValueError(f"Seller dataset not found: {seller_id}")
        
        logger.info(f"Fine-tuning model for {seller_id}...")
        
        # Initialize model
        model = FineTunedCLIP(self.config.CLIP_MODEL)
        model.to(self.device)
        
        # Create dataset
        dataset = ProductDataset(seller_path, model.processor)
        dataloader = DataLoader(dataset, batch_size=8, shuffle=True)
        
        # Setup training
        optimizer = torch.optim.AdamW(model.parameters(), lr=learning_rate)
        criterion = nn.CrossEntropyLoss()
        
        # Training loop
        model.train()
        for epoch in range(epochs):
            total_loss = 0
            for batch in dataloader:
                pixel_values = batch['pixel_values'].to(self.device)
                product_ids = batch['product_id'].to(self.device)
                
                optimizer.zero_grad()
                
                # Forward pass
                image_features, product_logits = model({'pixel_values': pixel_values})
                loss = criterion(product_logits, product_ids)
                
                # Backward pass
                loss.backward()
                optimizer.step()
                
                total_loss += loss.item()
            
            avg_loss = total_loss / len(dataloader)
            logger.info(f"Epoch {epoch+1}/{epochs}, Loss: {avg_loss:.4f}")
        
        # Save fine-tuned model
        model_path = self.config.MODELS_DIR / f"{seller_id}_finetuned.pt"
        torch.save(model.state_dict(), model_path)
        
        logger.info(f"Fine-tuned model saved: {model_path}")
        return model_path