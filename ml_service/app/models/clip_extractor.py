import torch
import numpy as np
from PIL import Image
from transformers import CLIPProcessor, CLIPModel
import os
from pathlib import Path

class CLIPExtractor:
    def __init__(self, model_name="openai/clip-vit-base-patch16"):
        self.model_name = model_name
        self.model = CLIPModel.from_pretrained(model_name)
        self.processor = CLIPProcessor.from_pretrained(model_name)
        
        # CPU optimizations
        if torch.cuda.is_available():
            self.model = self.model.half().cuda()
        else:
            # CPU optimizations
            torch.set_num_threads(4)  # Limit CPU threads
            self.model.eval()  # Set to eval mode
    
    def extract_embedding(self, image: Image.Image) -> np.ndarray:
        """Extract CLIP embedding from image with optimizations"""
        # Resize image for CPU optimization
        if image.size[0] > 224 or image.size[1] > 224:
            image = image.resize((224, 224), Image.Resampling.LANCZOS)
        
        inputs = self.processor(images=image, return_tensors="pt")
        
        # Move to GPU if available
        if torch.cuda.is_available():
            inputs = {k: v.cuda() for k, v in inputs.items()}
        
        with torch.no_grad():
            image_features = self.model.get_image_features(**inputs)
            # Normalize the features
            image_features = image_features / image_features.norm(dim=-1, keepdim=True)
            embedding = image_features.cpu().numpy().flatten()
        
        return embedding
    
    def extract_text_embedding(self, text: str) -> np.ndarray:
        """Extract text embedding for multimodal search"""
        text_inputs = self.processor(text=[text], return_tensors="pt")
        
        if torch.cuda.is_available():
            text_inputs = {k: v.cuda() for k, v in text_inputs.items()}
        
        with torch.no_grad():
            text_features = self.model.get_text_features(**text_inputs)
            text_features = text_features / text_features.norm(dim=-1, keepdim=True)
            embedding = text_features.cpu().numpy().flatten()
        
        return embedding