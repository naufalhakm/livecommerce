import torch
import os
from PIL import Image

class CPUOptimizer:
    @staticmethod
    def optimize_for_cpu():
        """Optimize PyTorch for CPU inference"""
        # Set optimal CPU threads
        torch.set_num_threads(min(4, os.cpu_count()))
        
        # Disable gradient computation globally
        torch.set_grad_enabled(False)
        
        # Set CPU optimization flags
        os.environ['OMP_NUM_THREADS'] = '4'
        os.environ['MKL_NUM_THREADS'] = '4'
    
    @staticmethod
    def resize_image_fast(image: Image.Image, size=(224, 224)) -> Image.Image:
        """Fast image resizing for CPU"""
        if image.size == size:
            return image
        return image.resize(size, Image.Resampling.BILINEAR)  # Faster than LANCZOS
    
    @staticmethod
    def batch_inference(model, inputs, batch_size=1):
        """Process in smaller batches for CPU"""
        with torch.no_grad():
            return model(**inputs)