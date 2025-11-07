from fastapi import FastAPI, File, UploadFile, HTTPException, BackgroundTasks, Form
from fastapi.responses import JSONResponse
import os
import time
import psutil
import subprocess
from models.yolo_detector import YOLODetector
from models.clip_extractor import CLIPExtractor
from models.faiss_index import FAISSIndex
from services.trainer import TrainerService
from utils.config import Config
from utils.cpu_optimizer import CPUOptimizer
from utils.metrics_logger import metrics_logger
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

# Initialize components with CPU optimization
CPUOptimizer.optimize_for_cpu()

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
async def train_model(seller_id: str, background_tasks: BackgroundTasks, fine_tune: bool = False):
    """Train model for all products from backend API"""
    try:
        # Use seller_X format for consistency
        seller_key = f"seller_{seller_id}"
        training_status[seller_key] = {"status": "training", "progress": 0, "message": "Starting training..."}
        
        if fine_tune:
            background_tasks.add_task(run_fine_tuning_pipeline, seller_key)
        else:
            background_tasks.add_task(run_training_pipeline, seller_key)
            
        return {"status": "training_started", "seller_id": seller_id, "seller_key": seller_key, "fine_tune": fine_tune}
    except Exception as e:
        seller_key = f"seller_{seller_id}"
        training_status[seller_key] = {"status": "error", "message": str(e)}
        raise HTTPException(status_code=500, detail=str(e))

async def run_training_pipeline(seller_key: str):
    """Run complete training pipeline"""
    try:
        training_status[seller_key] = {"status": "training", "progress": 20, "message": "Fetching products from API..."}
        
        # Organize dataset first
        if not trainer_service.organize_dataset_from_api():
            raise ValueError("Failed to organize dataset from API")
        
        training_status[seller_key] = {"status": "training", "progress": 50, "message": "Training model..."}
        
        # Train specific seller
        result = await trainer_service.train_seller_model(seller_key)
        
        training_status[seller_key] = {"status": "completed", "progress": 100, 
                                    "message": f"Training completed: {result['total_embeddings']} embeddings"}
    except Exception as e:
        logger.error(f"Training error for {seller_key}: {e}")
        training_status[seller_key] = {"status": "error", "progress": 0, "message": str(e)}

async def run_fine_tuning_pipeline(seller_key: str):
    """Run fine-tuning pipeline"""
    try:
        from services.fine_tuner import FineTuner
        
        training_status[seller_key] = {"status": "training", "progress": 10, "message": "Preparing fine-tuning..."}
        
        # Organize dataset first
        if not trainer_service.organize_dataset_from_api():
            raise ValueError("Failed to organize dataset from API")
        
        training_status[seller_key] = {"status": "training", "progress": 30, "message": "Fine-tuning CLIP model..."}
        
        # Fine-tune model
        fine_tuner = FineTuner(config)
        model_path = fine_tuner.fine_tune_seller_model(seller_key, epochs=5)
        
        training_status[seller_key] = {"status": "training", "progress": 80, "message": "Building optimized embeddings..."}
        
        # Train with fine-tuned model
        result = await trainer_service.train_seller_model(seller_key)
        
        training_status[seller_key] = {"status": "completed", "progress": 100, 
                                    "message": f"Fine-tuning completed: {result['total_embeddings']} embeddings"}
    except Exception as e:
        logger.error(f"Fine-tuning error for {seller_key}: {e}")
        training_status[seller_key] = {"status": "error", "progress": 0, "message": str(e)}

@app.post("/predict")
async def predict_products(seller_id: str = Form(...), file: UploadFile = File(...)):
    """Predict products in live stream frame - PRODUCTION ENDPOINT"""
    start_time = time.time()
    yolo_time = 0
    clip_time = 0
    
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
        
        # Track timing for each component
        yolo_start = time.time()
        result = await trainer_service.detect_products(seller_key, image_data)
        total_time = time.time() - start_time
        
        # Extract timing info from result if available
        yolo_time = result.get('yolo_time', 0.065)  # Default fallback
        clip_time = result.get('clip_time', 0.045)  # Default fallback
        
        # Log metrics
        detections = result.get('detections', [])
        predictions = result.get('predictions', [])
        
        # Log YOLO metrics
        metrics_logger.log_yolo_metrics(detections, yolo_time)
        
        # Log CLIP metrics
        similarity_scores = [p.get('similarity_score', 0) for p in predictions]
        product_matched = len([p for p in predictions if p.get('similarity_score', 0) > 0.8]) > 0
        metrics_logger.log_clip_metrics(similarity_scores, clip_time, predictions, product_matched)
        
        # Log system performance
        memory_usage = psutil.virtual_memory().used / 1024 / 1024  # MB
        cpu_usage = psutil.cpu_percent()
        fps = 1.0 / max(total_time, 0.001)
        metrics_logger.log_system_performance(total_time, total_time, fps, memory_usage, cpu_usage)
        
        logger.info(f"🎯 Detection result: {result}")
        
        return {
            'predictions': predictions,
            'detections': detections,
            'total_detections': len(detections),
            'total_products': len(predictions),
            'timing': {
                'yolo_time': yolo_time,
                'clip_time': clip_time,
                'total_time': total_time
            }
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
        "yolo_model": "yolo11n",
        "clip_model": "openai/clip-vit-base-patch32",
        "status": "active"
    }

@app.get("/training-status/{seller_id}")
async def get_training_status(seller_id: str):
    """Get training status for seller"""
    seller_key = f"seller_{seller_id}" if not seller_id.startswith('seller_') else seller_id
    return training_status.get(seller_key, {"status": "not_started"})

@app.get("/metrics")
async def get_metrics():
    """Get collected metrics data"""
    try:
        import json
        logs_dir = "experiment_logs"
        
        metrics = {}
        for filename in ['yolo_metrics.json', 'clip_metrics.json', 'system_metrics.json']:
            filepath = os.path.join(logs_dir, filename)
            if os.path.exists(filepath):
                with open(filepath, 'r') as f:
                    metrics[filename.replace('.json', '')] = json.load(f)
            else:
                metrics[filename.replace('.json', '')] = []
                
        return metrics
    except Exception as e:
        return {"error": str(e)}

@app.post("/generate-thesis-results")
async def generate_thesis_results():
    """Generate thesis charts and analysis"""
    try:
        import subprocess
        import json
        
        # Run evaluation script
        result = subprocess.run(
            ['python', 'evaluation_template.py'],
            capture_output=True,
            text=True,
            cwd='/app'
        )
        
        if result.returncode == 0:
            # Parse result
            try:
                output = json.loads(result.stdout.split('\n')[-2])  # Get last JSON line
                return {
                    "status": "success",
                    "message": "Thesis results generated successfully",
                    "results": output
                }
            except:
                return {
                    "status": "success", 
                    "message": "Charts generated",
                    "output": result.stdout
                }
        else:
            return {
                "status": "error",
                "message": "Failed to generate results",
                "error": result.stderr
            }
    except Exception as e:
        return {
            "status": "error",
            "message": str(e)
        }

@app.get("/health")
async def health_check():
    return {"status": "healthy"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8001)