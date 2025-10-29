FROM nvidia/cuda:13.0.1-cudnn-runtime-ubuntu22.04

# Install Python and system dependencies
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    libgl1-mesa-glx \
    libglib2.0-0 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install PyTorch with CUDA support
RUN pip3 install torch>=2.6.0 torchvision>=0.21.0 --index-url https://download.pytorch.org/whl/cu121

COPY requirements-gpu.txt .
RUN pip3 install --no-cache-dir -r requirements-gpu.txt

COPY . .

# Make training script executable
RUN chmod +x train_offline.py

# Pre-download CLIP model to avoid runtime download
RUN python3 -c "from transformers import CLIPModel, CLIPProcessor; CLIPModel.from_pretrained('openai/clip-vit-base-patch32'); CLIPProcessor.from_pretrained('openai/clip-vit-base-patch32')"

EXPOSE 8001

# Run training first, then start service
CMD ["sh", "-c", "python3 train_offline.py && python3 app/main.py"]