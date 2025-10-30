import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const api = axios.create({
  baseURL: `${API_BASE_URL}/api`,
});

export const productAPI = {
  getAll: () => api.get('/products'),
  getById: (id) => api.get(`/products/${id}`),
  create: (product) => api.post('/products', product),
  createWithFile: (formData) => {
    return axios.post(`${API_BASE_URL}/api/products`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    });
  },
  update: (id, product) => api.put(`/products/${id}`, product),
  addImages: (id, formData) => {
    return axios.post(`${API_BASE_URL}/api/products/${id}/images`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    });
  },
  delete: (id) => api.delete(`/products/${id}`),
  train: (id) => api.post(`/products/${id}/train`),
  predict: (id, imageFile) => {
    const formData = new FormData();
    formData.append('image', imageFile);
    return api.post(`/products/${id}/predict`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    });
  }
};

export const streamAPI = {
  getAll: () => api.get('/streams'),
  create: (stream) => api.post('/streams', stream),
  updateStatus: (id, isActive) => api.put(`/streams/${id}/status`, { is_active: isActive }),
  processFrame: (sellerId, frameFile) => {
    const formData = new FormData();
    formData.append('frame', frameFile);
    return api.post(`/stream/process-frame?seller_id=${sellerId}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    });
  },
  predictFrame: (sellerId, frameFile) => {
    const formData = new FormData();
    formData.append('frame', frameFile);
    return api.post(`/stream/predict?seller_id=${sellerId}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    });
  }
};

export const mlAPI = {
  trainModel: (sellerId) => api.post(`/products/train?seller_id=${sellerId}`),
  predictProduct: (sellerId, imageFile) => {
    const formData = new FormData();
    formData.append('frame', imageFile);
    return api.post(`/stream/predict?seller_id=${sellerId}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    });
  }
};

export const pinAPI = {
  pinProduct: (productId, sellerId, similarityScore) => {
    return api.post(`/products/${productId}/pin`, {
      seller_id: sellerId,
      similarity_score: similarityScore
    });
  },
  unpinProduct: (productId, sellerId) => {
    return api.delete(`/products/${productId}/unpin?seller_id=${sellerId}`);
  },
  getPinnedProducts: (sellerId) => api.get(`/products/pinned/${sellerId}`)
};

export default api;