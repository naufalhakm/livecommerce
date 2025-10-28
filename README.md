# Live Commerce AI System

A real-time product recognition system for live streaming commerce, similar to TikTok Live or Tokopedia Live.

## 🏗️ Architecture

- **Frontend (React)**: Live streaming interface and admin dashboard
- **Backend (Golang)**: WebSocket/WebRTC gateway, API CRUD, and data bridge
- **ML Service (Python FastAPI)**: Product recognition, embedding, and similarity search

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose
- Node.js 18+ (for local development)
- Go 1.21+ (for local development)
- Python 3.9+ (for local development)

### Run with Docker

```bash
# Clone and navigate to project
cd live-shopping-ai

# Start all services
docker-compose up --build

# Access the application
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
# ML Service: http://localhost:8001
```

### Local Development

#### 1. Database Setup
```bash
# Start PostgreSQL
docker run -d --name postgres \
  -e POSTGRES_DB=livecommerce \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 postgres:15
```

#### 2. ML Service
```bash
cd ml_service
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8001
```

#### 3. Backend
```bash
cd backend
go mod tidy
go run cmd/main.go
```

#### 4. Frontend
```bash
cd frontend
npm install
npm run dev
```

## 📁 Project Structure

```
live-shopping-ai/
├── ml_service/           # Python FastAPI ML service
│   ├── main.py          # FastAPI application
│   ├── requirements.txt # Python dependencies
│   └── Dockerfile       # ML service container
├── backend/             # Golang backend service
│   ├── cmd/main.go     # Application entry point
│   ├── internal/       # Internal packages
│   │   ├── api/        # HTTP handlers
│   │   ├── services/   # Business logic
│   │   ├── models/     # Database models
│   │   └── db/         # Database connection
│   ├── go.mod          # Go dependencies
│   └── Dockerfile      # Backend container
├── frontend/           # React frontend
│   ├── src/
│   │   ├── pages/      # React pages
│   │   ├── components/ # React components
│   │   └── services/   # API and WebSocket clients
│   ├── package.json    # Node dependencies
│   └── Dockerfile      # Frontend container
└── docker-compose.yml # Service orchestration
```

## 🔧 Configuration

### Environment Variables

**Backend:**
- `DB_HOST`: PostgreSQL host (default: localhost)
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_USER`: Database user (default: postgres)
- `DB_PASSWORD`: Database password (default: postgres)
- `DB_NAME`: Database name (default: livecommerce)
- `ML_SERVICE_URL`: ML service URL (default: http://localhost:8001)

**Frontend:**
- `VITE_API_URL`: Backend API URL (default: http://localhost:8080)
- `VITE_WS_URL`: WebSocket URL (default: ws://localhost:8080)

## 📊 Dataset Structure

Create datasets for training:

```
datasets/
├── seller_1/
│   ├── iphone_15/
│   │   ├── image1.jpg
│   │   └── image2.jpg
│   └── macbook_pro/
│       ├── image1.jpg
│       └── image2.jpg
└── seller_2/
    └── samsung_phone/
        ├── image1.jpg
        └── image2.jpg
```

## 🎯 Usage

### 1. Admin Dashboard
- Navigate to `/admin`
- Add products with details
- Upload product images
- Train ML models per seller

### 2. Live Streaming
- Navigate to `/` (home)
- Start streaming with camera
- Products are automatically detected
- Real-time product overlay on video

### 3. API Endpoints

**Products:**
- `GET /api/products` - List all products
- `POST /api/products` - Create product
- `POST /api/products/:id/train` - Train model
- `POST /api/products/:id/predict` - Predict product

**Streams:**
- `GET /api/streams` - List streams
- `POST /api/streams` - Create stream
- `WS /ws/livestream` - WebSocket connection

**ML Service:**
- `POST /train?seller_id=X` - Train model
- `POST /predict` - Predict products
- `GET /model-info` - Model information

## 🔍 Features

- **Real-time Product Detection**: YOLO-based object detection
- **Product Recognition**: CLIP embeddings with FAISS similarity search
- **Live Streaming**: WebRTC-based video streaming
- **WebSocket Communication**: Real-time updates
- **Admin Dashboard**: Product and model management
- **Multi-seller Support**: Separate models per seller

## 🛠️ Technology Stack

| Component | Technology |
|-----------|------------|
| Frontend | React, Vite, Tailwind CSS, WebRTC |
| Backend | Golang, Gin, GORM, WebSocket |
| ML Service | Python, FastAPI, PyTorch, YOLO, CLIP, FAISS |
| Database | PostgreSQL |
| Containerization | Docker, Docker Compose |

## 📝 Development Notes

- Models are automatically downloaded on first run
- FAISS indices are saved per seller
- WebRTC signaling handled via WebSocket
- Frame processing occurs every 2 seconds during streaming
- All services include health check endpoints

## 🚨 Production Considerations

- Add authentication and authorization
- Implement rate limiting
- Use cloud storage for datasets and models
- Add monitoring and logging
- Configure HTTPS and WSS
- Implement horizontal scaling
- Add caching layer (Redis)
- Use message queue for async processing