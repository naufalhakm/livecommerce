# Server Deployment Guide

## Prerequisites

### Server Requirements
- Ubuntu 20.04+ or similar Linux distribution
- NVIDIA GPU with CUDA support
- Docker and Docker Compose
- Git

### Install CUDA (Ubuntu)
```bash
# Install NVIDIA drivers
sudo apt update
sudo apt install nvidia-driver-470

# Install CUDA toolkit
wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2004/x86_64/cuda-ubuntu2004.pin
sudo mv cuda-ubuntu2004.pin /etc/apt/preferences.d/cuda-repository-pin-600
wget https://developer.download.nvidia.com/compute/cuda/11.8.0/local_installers/cuda-repo-ubuntu2004-11-8-local_11.8.0-520.61.05-1_amd64.deb
sudo dpkg -i cuda-repo-ubuntu2004-11-8-local_11.8.0-520.61.05-1_amd64.deb
sudo cp /var/cuda-repo-ubuntu2004-11-8-local/cuda-*-keyring.gpg /usr/share/keyrings/
sudo apt-get update
sudo apt-get -y install cuda

# Verify installation
nvidia-smi
nvcc --version
```

## Deployment Steps

### 1. Clone Repository
```bash
git clone https://github.com/yourusername/live-shopping-ai.git
cd live-shopping-ai
```

### 2. Environment Setup
```bash
# Copy environment files
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env

# Edit environment variables
nano backend/.env
nano frontend/.env
```

### 3. Build and Run with Docker
```bash
# Build all services
docker-compose build

# Start services
docker-compose up -d

# Check logs
docker-compose logs -f
```

### 4. Manual Setup (Alternative)

#### Backend (Go)
```bash
cd backend
go mod tidy
go build -o main cmd/main.go
./main
```

#### Frontend (React)
```bash
cd frontend
npm install
npm run build
npm run preview
```

#### ML Service (Python with CUDA)
```bash
cd ml_service
pip3 install torch torchvision --index-url https://download.pytorch.org/whl/cu118
pip3 install -r requirements.txt
python3 app/main.py
```

### 5. Database Setup
```bash
# Start PostgreSQL
docker run -d --name postgres \
  -e POSTGRES_DB=livecommerce \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 postgres:15
```

## Service URLs
- Frontend: http://your-server:7000
- Backend API: http://your-server:7080
- ML Service: http://your-server:7001

## Troubleshooting

### CUDA Issues
```bash
# Check CUDA availability in Python
python3 -c "import torch; print(torch.cuda.is_available())"
```

### Port Issues
```bash
# Check if ports are in use
sudo netstat -tulpn | grep :7000
sudo netstat -tulpn | grep :7080
sudo netstat -tulpn | grep :7001
```

### Docker Issues
```bash
# Restart Docker
sudo systemctl restart docker

# Clean up
docker system prune -a
```