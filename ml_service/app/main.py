from fastapi import FastAPI, File, UploadFile, HTTPException, BackgroundTasks, Form
from fastapi.responses import JSONResponse
import os
from models.yolo_detector import YOLODetector
from models.clip_extractor import CLIPExtractor
from models.faiss_index import FAISSIndex
from services.trainer import TrainerService
from utils.config import Config
import logging

import sys
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)

app = FastAPI(title="Live Commerce ML Service")

# Initialize components
config = Config()
yolo_detector = YOLODetector()
clip_extractor = CLIPExtractor()
faiss_index = FAISSIndex(config.EMBEDDINGS_DIR)
trainer_service = TrainerService(yolo_detector, clip_extractor, faiss_index, config)

logger.info(f"ML Service initialized. Available sellers: {faiss_index.get_loaded_sellers()}")

# Training status tracking
training_status = {}

# Don't run training on startup - wait for API call

@app.post("/train")
async def train_model(seller_id: str, background_tasks: BackgroundTasks):
    """Train model for all products from backend API"""
    try:
        # Use seller_X format for consistency
        seller_key = f"seller_{seller_id}"
        training_status[seller_key] = {"status": "training", "progress": 0, "message": "Starting training..."}
        background_tasks.add_task(run_training_pipeline, seller_key)
        return {"status": "training_started", "seller_id": seller_id}
    except Exception as e:
        seller_key = f"seller_{seller_id}"
        training_status[seller_key] = {"status": "error", "message": str(e)}
        raise HTTPException(status_code=500, detail=str(e))

async def run_training_pipeline(seller_key: str):
    """Run complete training pipeline"""
    try:
        training_status[seller_key] = {"status": "training", "progress": 20, "message": "Fetching products from API..."}
        
        # Train all sellers from API
        results = await trainer_service.train_all_sellers()
        
        training_status[seller_key] = {"status": "completed", "progress": 100, 
                                    "message": f"Training completed for all sellers: {list(results.keys())}"}
    except Exception as e:
        training_status[seller_key] = {"status": "error", "progress": 0, "message": str(e)}

@app.post("/predict")
async def predict_products(seller_id: str = Form(...), file: UploadFile = File(...)):
    """Predict products in live stream frame - PRODUCTION ENDPOINT"""
    try:
        print(f"🔍 PREDICT REQUEST: seller_id={seller_id}, file={file.filename}")
        logger.info(f"🔍 PREDICT REQUEST: seller_id={seller_id}, file={file.filename}")
        
        # Convert seller_id to seller_X format for FAISS lookup
        seller_key = f"seller_{seller_id}"
        logger.info(f"🔑 Converted to seller_key: {seller_key}")
        logger.info(f"📊 Available sellers: {list(faiss_index.seller_indices.keys())}")
        
        if seller_key not in faiss_index.seller_indices:
            logger.warning(f"❌ No model found for {seller_key}")
            return {
                'predictions': [],
                'total_detections': 0,
                'message': f'No trained model found for seller {seller_id}. Please train first.'
            }
        
        logger.info(f"✅ Model found for {seller_key}")
        image_data = await file.read()
        logger.info(f"📷 Image size: {len(image_data)} bytes")
        
        result = await trainer_service.detect_products(seller_key, image_data)
        logger.info(f"🎯 Detection result: {result}")
        
        return {
            'predictions': result.get('predictions', []),
            'total_detections': len(result.get('predictions', []))
        }
    except Exception as e:
        logger.error(f"❌ Prediction error: {e}")
        import traceback
        logger.error(f"📋 Traceback: {traceback.format_exc()}")
        return {
            'predictions': [],
            'total_detections': 0,
            'error': str(e)
        }

@app.post("/detect-live")
async def detect_products_live(seller_id: str, file: UploadFile = File(...)):
    """Detect products in live stream frame with auto-pin to cart"""
    try:
        image_data = await file.read()
        result = await trainer_service.detect_products(seller_id, image_data)
        
        # Auto-pin high confidence products to cart
        backend_url = os.getenv('BACKEND_URL', 'http://backend:8080')
        pinned_products = []
        
        for prediction in result.get('predictions', []):
            if prediction['similarity_score'] > 0.8:  # High confidence threshold
                try:
                    import requests
                    # Send to backend API to pin product
                    pin_response = requests.post(
                        f"{backend_url}/api/products/{prediction['product_id']}/pin",
                        json={'seller_id': seller_id, 'similarity_score': prediction['similarity_score']}
                    )
                    if pin_response.status_code == 200:
                        pinned_products.append(prediction['product_id'])
                except Exception as e:
                    logger.error(f"Failed to pin product {prediction['product_id']}: {e}")
        
        return {
            'predictions': result.get('predictions', []),
            'pinned_products': pinned_products,
            'message': f"Detected {len(result.get('predictions', []))} objects, pinned {len(pinned_products)} products"
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/reload-models")
async def reload_models():
    """Reload all trained models"""
    try:
        faiss_index._load_existing_indices()
        return {"status": "success", "message": "All models reloaded", 
                "available_sellers": faiss_index.get_loaded_sellers()}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/model-info")
async def get_model_info():
    """Get model information"""
    return {
        "loaded_sellers": faiss_index.get_loaded_sellers(),
        "total_vectors": {seller: idx.ntotal for seller, idx in faiss_index.seller_indices.items()},
        "yolo_model": "yolov8n",
        "clip_model": "openai/clip-vit-base-patch32",
        "status": "active"
    }

@app.get("/training-status/{seller_id}")
async def get_training_status(seller_id: str):
    """Get training status for seller"""
    seller_key = f"seller_{seller_id}" if not seller_id.startswith('seller_') else seller_id
    return training_status.get(seller_key, {"status": "not_started"})

@app.get("/health")
async def health_check():
    return {"status": "healthy"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8001)