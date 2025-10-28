import io from 'socket.io-client';

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080';

class WebSocketService {
  constructor() {
    this.socket = null;
    this.listeners = new Map();
  }

  connect(clientId, roomId) {
    if (!clientId || !roomId) {
      console.error('client_id and room_id are required');
      return;
    }

    if (this.socket) {
      this.disconnect();
    }

    this.socket = new WebSocket(`${WS_URL}/ws/livestream?client_id=${clientId}&room_id=${roomId}`);
    
    this.socket.onopen = () => {
      console.log(`WebSocket connected to room ${roomId}`);
      this.emit('connected');
    };

    this.socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        this.emit(message.type, message);
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };

    this.socket.onclose = () => {
      console.log('WebSocket disconnected');
      this.emit('disconnected');
    };

    this.socket.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.emit('error', error);
    };
  }

  disconnect() {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }

  send(message) {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    }
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event).push(callback);
  }

  off(event, callback) {
    if (this.listeners.has(event)) {
      const callbacks = this.listeners.get(event);
      const index = callbacks.indexOf(callback);
      if (index > -1) {
        callbacks.splice(index, 1);
      }
    }
  }

  emit(event, data) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).forEach(callback => callback(data));
    }
  }

  // WebRTC signaling methods
  sendOffer(offer, to) {
    this.send({
      type: 'webrtc_offer',
      data: offer,
      to: to
    });
  }

  sendAnswer(answer, to) {
    this.send({
      type: 'webrtc_answer',
      data: answer,
      to: to
    });
  }

  sendIceCandidate(candidate, to) {
    this.send({
      type: 'webrtc_ice_candidate',
      data: candidate,
      to: to
    });
  }

  // Chat method
  sendChat(message, username) {
    this.send({
      type: 'chat',
      data: {
        message: message,
        username: username,
        timestamp: new Date().toISOString()
      }
    });
  }
}

export default new WebSocketService();