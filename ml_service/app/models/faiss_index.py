import faiss
import numpy as np
import json
import os
from pathlib import Path
import logging

logger = logging.getLogger(__name__)

class FAISSIndex:
    def __init__(self, embeddings_dir):
        self.embeddings_dir = Path(embeddings_dir)
        self.seller_indices = {}
        self.seller_products = {}
        self._load_existing_indices()
    
    def _load_existing_indices(self):
        """Load existing FAISS indices for all sellers"""
        for file in self.embeddings_dir.glob("*_faiss.index"):
            seller_id = file.stem.replace("_faiss", "")
            try:
                self.load_seller_index(seller_id)
                logger.info(f"Loaded index for seller: {seller_id}")
            except Exception as e:
                logger.error(f"Error loading index for {seller_id}: {e}")
    
    def load_seller_index(self, seller_id: str):
        """Load FAISS index and product mapping for seller"""
        index_path = self.embeddings_dir / f"{seller_id}_faiss.index"
        products_path = self.embeddings_dir / f"{seller_id}_products.json"
        
        if index_path.exists():
            self.seller_indices[seller_id] = faiss.read_index(str(index_path))
            
            if products_path.exists():
                with open(products_path, 'r') as f:
                    self.seller_products[seller_id] = json.load(f)
    
    def save_seller_index(self, seller_id: str, embeddings: np.ndarray, product_metadata: list):
        """Save FAISS index and product mapping for seller"""
        # Create FAISS index
        embeddings_array = embeddings.astype('float32')
        index = faiss.IndexFlatIP(embeddings_array.shape[1])
        index.add(embeddings_array)
        
        # Save index and metadata
        index_path = self.embeddings_dir / f"{seller_id}_faiss.index"
        products_path = self.embeddings_dir / f"{seller_id}_products.json"
        
        faiss.write_index(index, str(index_path))
        with open(products_path, 'w') as f:
            json.dump(product_metadata, f)
        
        self.seller_indices[seller_id] = index
        self.seller_products[seller_id] = product_metadata
    
    def search(self, seller_id: str, query_embedding: np.ndarray, k: int = 1):
        """Search for similar products"""
        if seller_id not in self.seller_indices:
            raise ValueError(f"No index found for seller: {seller_id}")
        
        query_embedding = query_embedding.reshape(1, -1).astype('float32')
        scores, indices = self.seller_indices[seller_id].search(query_embedding, k)
        
        results = []
        for i, (score, idx) in enumerate(zip(scores[0], indices[0])):
            if idx < len(self.seller_products[seller_id]):
                product_info = self.seller_products[seller_id][idx]
                results.append({
                    "product_id": product_info["product_id"],
                    "product_name": product_info["product_name"],
                    "similarity_score": float(score)
                })
        
        return results
    
    def get_loaded_sellers(self):
        """Get list of loaded seller IDs"""
        return list(self.seller_indices.keys())