# Docker Setup Guide

This project supports multiple Docker configurations for different environments:

## 🖥️ Local Development (macOS/ARM64)

For local development on macOS with Apple Silicon:

```bash
# Use CPU-optimized configuration
docker compose -f docker-compose.cpu.yml up -d

# Access services:
# - Frontend: http://localhost:3000
# - Backend API: http://localhost:8080
# - ML Service: http://localhost:8001
```

## 🚀 Production/GPU Environment

For production deployment with NVIDIA GPU support:

```bash
# Use GPU-optimized configuration
docker compose -f docker-compose.gpu.yml up -d

# Or use the main configuration
docker compose up -d
```

## 📁 Configuration Files

- `docker-compose.yml` - Main configuration (GPU-enabled)
- `docker-compose.gpu.yml` - Explicit GPU configuration
- `docker-compose.cpu.yml` - CPU-only configuration for local development

## 🔧 ML Service Configurations

- `ml_service/Dockerfile` - Production (NVIDIA CUDA + GPU)
- `ml_service/Dockerfile.cpu` - Local development (CPU-only)
- `ml_service/requirements.txt` - Production dependencies (faiss-gpu)
- `ml_service/requirements-cpu.txt` - Local development dependencies (faiss-cpu)

## 🛠️ Troubleshooting

### Port Conflicts
If you encounter port conflicts, modify the ports in the respective docker-compose file:
- Frontend: Change `3000:3000` to `<available-port>:3000`
- Backend: Change `8080:8080` to `<available-port>:8080`
- ML Service: Change `8001:8001` to `<available-port>:8001`

### GPU Issues
If you don't have NVIDIA GPU support, always use the CPU configuration:
```bash
docker compose -f docker-compose.cpu.yml up -d
```