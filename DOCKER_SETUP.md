# 🐳 Docker Setup untuk Thesis Metrics System

## 🚀 Quick Start

### 1. Build dan Start Services
```bash
# Clone project (jika belum)
git clone <your-repo>
cd live-shopping-ai

# Build dan start semua services
docker-compose up --build

# Atau run di background
docker-compose up --build -d
```

### 2. Access Applications
```
Frontend:     http://localhost:3000
Backend API:  http://localhost:8080  
ML Service:   http://localhost:8001
Metrics:      http://localhost:3000/metrics
```

### 3. Generate Thesis Results
1. Buka `http://localhost:3000/metrics`
2. Klik **"🎓 Generate Thesis Results"**
3. Charts akan dibuat di container ML service
4. Download via frontend

## 📁 Volume Mappings

```yaml
ml_service:
  volumes:
    - ./datasets:/app/datasets              # Training data
    - ./embeddings:/app/embeddings          # FAISS indices  
    - ./models:/app/models                  # YOLO models
    - ./ml_service/experiment_logs:/app/experiment_logs    # Real metrics data
    - ./ml_service/thesis_results:/app/thesis_results      # Generated charts
```

## 🔧 Environment Variables

### Backend
```yaml
DATABASE_URL: postgresql://...              # Supabase DB
ML_SERVICE_URL: http://ml_service:8001      # Internal ML service
```

### Frontend  
```yaml
VITE_API_URL: https://livecomerce.laidtechnology.tech     # Production API
VITE_WS_URL: wss://livecomerce.laidtechnology.tech        # Production WS
```

### ML Service
```yaml
BACKEND_URL: http://backend:8080            # Internal backend
PYTHONUNBUFFERED: 1                         # Python logging
```

## 📊 Data Flow dalam Docker

```
Live Streaming → ML Service Container → experiment_logs/ → Backend Container → Frontend Container
                      ↓
                thesis_results/ (charts PNG + analysis JSON)
```

## 🛠️ Development Commands

### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f ml_service
docker-compose logs -f backend
docker-compose logs -f frontend
```

### Restart Services
```bash
# Restart specific service
docker-compose restart ml_service

# Rebuild and restart
docker-compose up --build ml_service
```

### Access Container Shell
```bash
# ML Service container
docker-compose exec ml_service bash

# Backend container  
docker-compose exec backend sh

# Check generated files
docker-compose exec ml_service ls -la thesis_results/
```

### Stop Services
```bash
# Stop all
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## 📈 Thesis Results Location

### Inside Container:
```
/app/experiment_logs/          # Real metrics JSON
/app/thesis_results/           # Generated charts PNG
```

### On Host Machine:
```
./ml_service/experiment_logs/  # Real metrics JSON  
./ml_service/thesis_results/   # Generated charts PNG
```

## 🔍 Troubleshooting

### 1. ML Service Won't Start
```bash
# Check Python dependencies
docker-compose exec ml_service pip list

# Check if evaluation script exists
docker-compose exec ml_service ls -la evaluation_template.py
```

### 2. Charts Not Generated
```bash
# Check ML service logs
docker-compose logs ml_service

# Test evaluation script manually
docker-compose exec ml_service python evaluation_template.py
```

### 3. No Metrics Data
```bash
# Check if experiment_logs exists
docker-compose exec ml_service ls -la experiment_logs/

# Test live streaming to generate data
# Go to http://localhost:3000/seller and start streaming
```

### 4. Frontend Can't Access Backend
```bash
# Check network connectivity
docker-compose exec frontend ping backend
docker-compose exec frontend ping ml_service
```

## 📝 Production Deployment

### 1. Update Environment Variables
```yaml
# docker-compose.prod.yml
frontend:
  environment:
    - VITE_API_URL=https://your-domain.com
    - VITE_WS_URL=wss://your-domain.com
```

### 2. Use Production Build
```bash
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up --build
```

### 3. Setup Reverse Proxy (Nginx)
```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://localhost:3000;
    }
    
    location /api {
        proxy_pass http://localhost:8080;
    }
}
```

## ✅ Verification Steps

1. **Services Running:**
   ```bash
   docker-compose ps
   ```

2. **Health Checks:**
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8001/health  
   curl http://localhost:3000
   ```

3. **Generate Test Data:**
   - Start live streaming di `/seller`
   - Check metrics di `/metrics`
   - Generate thesis results

4. **Check Generated Files:**
   ```bash
   ls -la ml_service/thesis_results/
   ```

Sekarang sistem lengkap berjalan di Docker dengan thesis metrics generation! 🎓