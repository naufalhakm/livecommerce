import torch
import torch.nn as nn
from transformers import CLIPModel, CLIPProcessor
import numpy as np
from PIL import Image

class FineTunedCLIP(nn.Module):
    def __init__(self, model_name="openai/clip-vit-large-patch14-336", num_products=100):
        super().__init__()
        self.clip = CLIPModel.from_pretrained(model_name)
        self.processor = CLIPProcessor.from_pretrained(model_name)
        
        # Add product-specific classification head
        self.product_classifier = nn.Sequential(
            nn.Linear(self.clip.config.projection_dim, 512),
            nn.ReLU(),
            nn.Dropout(0.1),
            nn.Linear(512, num_products)
        )
        
        # Freeze CLIP backbone initially
        for param in self.clip.parameters():
            param.requires_grad = False
    
    def unfreeze_clip(self):
        """Unfreeze CLIP for fine-tuning"""
        for param in self.clip.parameters():
            param.requires_grad = True
    
    def extract_embedding(self, image: Image.Image) -> np.ndarray:
        """Extract enhanced embedding"""
        inputs = self.processor(images=image, return_tensors="pt")
        with torch.no_grad():
            image_features = self.clip.get_image_features(**inputs)
            embedding = image_features.cpu().numpy().flatten()
        return embedding / np.linalg.norm(embedding)
    
    def forward(self, images, texts=None):
        """Forward pass for training"""
        image_features = self.clip.get_image_features(**images)
        
        if texts is not None:
            text_features = self.clip.get_text_features(**texts)
            return image_features, text_features
        
        # Product classification
        product_logits = self.product_classifier(image_features)
        return image_features, product_logits