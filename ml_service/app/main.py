from fastapi import FastAPI, File, UploadFile, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse
import os
from models.yolo_detector import YOLODetector
from models.clip_extractor import CLIPExtractor
from models.faiss_index import FAISSIndex
from services.trainer import TrainerService
from utils.config import Config
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Live Commerce ML Service")

# Initialize components
config = Config()
yolo_detector = YOLODetector()
clip_extractor = CLIPExtractor()
faiss_index = FAISSIndex(config.EMBEDDINGS_DIR)
trainer_service = TrainerService(yolo_detector, clip_extractor, faiss_index, config)

@app.post("/train")
async def train_model(seller_id: str, background_tasks: BackgroundTasks):
    """Train model for specific seller using product_id structure"""
    try:
        training_status[seller_id] = {"status": "training", "progress": 0, "message": "Starting training..."}
        background_tasks.add_task(train_with_progress, seller_id)
        return {"status": "training_started", "seller_id": seller_id}
    except Exception as e:
        training_status[seller_id] = {"status": "error", "message": str(e)}
        raise HTTPException(status_code=500, detail=str(e))

async def train_with_progress(seller_id: str):
    """Train model with progress updates"""
    try:
        training_status[seller_id] = {"status": "training", "progress": 10, "message": "Loading datasets..."}
        result = await trainer_service.train_seller_model(seller_id)
        training_status[seller_id] = {"status": "completed", "progress": 100, "message": f"Training completed! {result['total_embeddings']} embeddings processed."}
    except Exception as e:
        training_status[seller_id] = {"status": "error", "progress": 0, "message": str(e)}

@app.post("/detect")
async def detect_products(seller_id: str, file: UploadFile = File(...)):
    """Detect products in uploaded image"""
    try:
        image_data = await file.read()
        result = await trainer_service.detect_products(seller_id, image_data)
        return JSONResponse(content=result)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/reload-model")
async def reload_model(seller_id: str):
    """Reload trained model for seller"""
    try:
        faiss_index.load_seller_index(seller_id)
        return {"status": "success", "message": f"Model reloaded for seller: {seller_id}"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/model-info")
async def get_model_info():
    """Get model information"""
    return {
        "loaded_sellers": faiss_index.get_loaded_sellers(),
        "yolo_model": "yolov8n",
        "clip_model": "openai/clip-vit-base-patch32",
        "status": "active"
    }

# Training status tracking
training_status = {}

@app.get("/training-status/{seller_id}")
async def get_training_status(seller_id: str):
    """Get training status for seller"""
    return training_status.get(seller_id, {"status": "not_started"})

@app.get("/health")
async def health_check():
    return {"status": "healthy"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8001)