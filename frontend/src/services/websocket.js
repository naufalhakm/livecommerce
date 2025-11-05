const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080';

class WebSocketService {
  constructor() {
    this.socket = null;
    this.listeners = new Map();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000;
    this.isConnecting = false;
    this.connectionCallbacks = {
      onConnected: null,
      onDisconnected: null,
      onError: null,
    };
  }

  connect(clientId, roomId) {
    if (!clientId || !roomId) {
      return;
    }

    if (this.isConnecting) {
      return;
    }

    this.isConnecting = true;
    this.clientId = clientId;
    this.roomId = roomId;

    // Clean up existing connection
    if (this.socket) {
      this.disconnect();
    }

    try {
      const url = `${WS_URL}/ws/livestream`;
      
      this.socket = new WebSocket(url);
      
      this.socket.onopen = () => {
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        
        // Send join message
        this.send({
          type: 'join',
          room: roomId,
          data: {
            client_id: clientId,
            role: clientId.includes('seller') ? 'publisher' : 'viewer'
          }
        });
        
        this.emit('connected', { clientId, roomId });
        
        if (this.connectionCallbacks.onConnected) {
          this.connectionCallbacks.onConnected();
        }
      };

      this.socket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.emit(message.type, message);
        } catch (error) {
        }
      };

      this.socket.onclose = (event) => {
        this.isConnecting = false;
        this.emit('disconnected', { code: event.code, reason: event.reason });
        
        if (this.connectionCallbacks.onDisconnected) {
          this.connectionCallbacks.onDisconnected(event);
        }

        // Attempt reconnection for unexpected closures
        if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.attemptReconnect(clientId, roomId);
        }
      };

      this.socket.onerror = (error) => {
        this.isConnecting = false;
        this.emit('error', error);
        
        if (this.connectionCallbacks.onError) {
          this.connectionCallbacks.onError(error);
        }
      };

    } catch (error) {
      this.isConnecting = false;
      this.emit('error', error);
    }
  }

  attemptReconnect(clientId, roomId) {
    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    
    
    setTimeout(() => {
      if (this.reconnectAttempts <= this.maxReconnectAttempts) {
        this.connect(clientId, roomId);
      } else {
        this.emit('reconnection_failed');
      }
    }, delay);
  }

  disconnect() {
    if (this.socket) {
      this.socket.onopen = null;
      this.socket.onmessage = null;
      this.socket.onclose = null;
      this.socket.onerror = null;
      this.socket.close(1000, 'Manual disconnect');
      this.socket = null;
    }
    this.isConnecting = false;
    this.reconnectAttempts = 0;
    this.listeners.clear();
  }

  send(message) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return false;
    }

    try {
      // Add room and client_id if not present
      if (!message.room && this.roomId) {
        message.room = this.roomId;
      }
      if (!message.client_id && this.clientId) {
        message.client_id = this.clientId;
      }
      
      this.socket.send(JSON.stringify(message));
      return true;
    } catch (error) {
      return false;
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
      this.listeners.get(event).forEach((callback, index) => {
        try {
          callback(data);
        } catch (error) {
        }
      });
    } else {
    }
  }

  // Connection lifecycle callbacks
  setConnectionCallbacks({ onConnected, onDisconnected, onError }) {
    this.connectionCallbacks = {
      onConnected,
      onDisconnected,
      onError,
    };
  }

  // WebRTC signaling methods
  sendOffer(offer, to) {
    return this.send({
      type: 'webrtc_offer',
      room: this.roomId,
      data: offer,
      from: this.clientId,
      to: to
    });
  }

  sendAnswer(answer, to) {
    return this.send({
      type: 'webrtc_answer',
      room: this.roomId,
      data: answer,
      from: this.clientId,
      to: to
    });
  }

  sendIceCandidate(candidate, to) {
    return this.send({
      type: 'webrtc_ice_candidate',
      room: this.roomId,
      data: candidate,
      from: this.clientId,
      to: to
    });
  }

  // Chat method
  sendChat(message, username) {
    return this.send({
      type: 'chat',
      data: {
        message: message,
        username: username,
        timestamp: new Date().toISOString()
      }
    });
  }

  // Get connection status
  getConnectionStatus() {
    if (!this.socket) return 'disconnected';
    
    switch (this.socket.readyState) {
      case WebSocket.CONNECTING:
        return 'connecting';
      case WebSocket.OPEN:
        return 'connected';
      case WebSocket.CLOSING:
        return 'closing';
      case WebSocket.CLOSED:
        return 'disconnected';
      default:
        return 'unknown';
    }
  }
}

export default new WebSocketService();