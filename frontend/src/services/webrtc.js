import websocketService from './websocket';

class WebRTCService {
  constructor() {
    this.peers = new Map();
    this.localStream = null;
    this.remoteStream = null;
    this.signalListenersSetup = false;
    this.SimplePeer = null;
    this.connectionState = 'disconnected';
    this.statsInterval = null;
    this.iceProcessingEnabled = true; // Added class-level property
  }

  async loadSimplePeer() {
    if (!this.SimplePeer) {
      if (window.SimplePeer) {
        this.SimplePeer = window.SimplePeer;
      } else {
        await this.loadSimplePeerFromCDN();
      }
    }
    return this.SimplePeer;
  }

  async loadSimplePeerFromCDN() {
    return new Promise((resolve, reject) => {
      if (window.SimplePeer) {
        this.SimplePeer = window.SimplePeer;
        resolve(this.SimplePeer);
        return;
      }

      const script = document.createElement('script');
      script.src = 'https://cdn.jsdelivr.net/npm/simple-peer@9.11.1/simplepeer.min.js';
      script.onload = () => {
        this.SimplePeer = window.SimplePeer;
        resolve(this.SimplePeer);
      };
      script.onerror = () => {
        reject(new Error('Failed to load SimplePeer from CDN'));
      };
      document.head.appendChild(script);
    });
  }

  async initializeCamera(facingMode = 'user') {
    try {
      if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
        throw new Error('Camera access not supported in this browser');
      }

      const constraints = {
        video: { 
          width: { ideal: 1280, max: 1280 }, 
          height: { ideal: 720, max: 720 },
          frameRate: { ideal: 30, max: 30 },
          facingMode: facingMode
        },
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          sampleRate: 44100,
          channelCount: 1
        }
      };

      this.localStream = await navigator.mediaDevices.getUserMedia(constraints);

      this.localStream.getTracks().forEach(track => {
        track.applyConstraints(constraints);
      });

      return this.localStream;
    } catch (error) {
      
      if (error.name === 'NotAllowedError') {
        throw new Error('Camera permission denied. Please allow camera access and try again.');
      } else if (error.name === 'NotFoundError') {
        throw new Error('No camera found. Please check your device has a camera.');
      } else if (error.name === 'NotReadableError') {
        throw new Error('Camera is already in use by another application.');
      } else {
        throw error;
      }
    }
  }
  
  async initializeScreenShare() {
    try {
      if (!navigator.mediaDevices || !navigator.mediaDevices.getDisplayMedia) {
        throw new Error('Screen sharing not supported in this browser');
      }

      this.localStream = await navigator.mediaDevices.getDisplayMedia({
        video: { 
          cursor: 'always',
          displaySurface: 'window'
        },
        audio: {
          echoCancellation: true,
          noiseSuppression: true
        }
      });

      this.localStream.getVideoTracks()[0].addEventListener('ended', () => {
        this.emitScreenShareEnded();
      });

      return this.localStream;
    } catch (error) {
      throw error;
    }
  }

  emitScreenShareEnded() {
    if (this.onScreenShareEnded) {
      this.onScreenShareEnded();
    }
  }

  async createPeer(isInitiator = false, targetClientId = null) {
    
    const peerId = targetClientId || `peer-${Date.now()}`;
    
    await this.cleanupPeer(peerId);

    const SimplePeer = await this.loadSimplePeer();

    const peerConfig = {
      initiator: isInitiator,
      trickle: true,
      config: {
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' },
          { urls: 'stun:stun1.l.google.com:19302' },
          { urls: 'stun:stun2.l.google.com:19302' },
          { urls: 'stun:stun3.l.google.com:19302' }
        ]
      },
      iceTransportPolicy: 'all',
      reconnectTimer: 1000,
    };
    
    if (isInitiator) {
      peerConfig.offerOptions = {
        offerToReceiveAudio: true,
        offerToReceiveVideo: true
      };
    }

    if (this.localStream && !isInitiator) {
      peerConfig.stream = this.localStream;
    } else if (this.localStream && isInitiator) {
      // Viewer should not send stream, only receive
    }


    let peer;
    try {
      peer = new SimplePeer(peerConfig);
    } catch (error) {
      throw error;
    }

    this.peers.set(peerId, peer);
    
    this.setupPeerHandlers(peer, peerId, isInitiator);
    
    return peer;
  }

  async cleanupPeer(peerId) {
    if (this.peers.has(peerId)) {
      const existingPeer = this.peers.get(peerId);
      try {
        existingPeer.destroy();
      } catch (error) {
      }
      this.peers.delete(peerId);
    }
  }

  setupPeerHandlers(peer, peerId, isInitiator) {
    let offerSent = false;
    let answerSent = false;
    let stateInterval;
    let connectionHealthy = false;
    let iceErrorCount = 0;
    const maxIceErrors = 5;

    peer.on('signal', (data) => {
      if (data.type === 'offer' && !offerSent) {
        offerSent = true;
        websocketService.send({
          type: 'webrtc_offer',
          data: {
            type: 'offer',
            sdp: data.sdp
          },
          from: websocketService.clientId,
          to: peerId
        });
      } else if (data.type === 'answer' && !answerSent) {
        answerSent = true;
        websocketService.send({
          type: 'webrtc_answer',
          data: {
            type: 'answer',
            sdp: data.sdp
          },
          from: websocketService.clientId,
          to: peerId
        });
      } else if (data.candidate && !connectionHealthy) {
        websocketService.send({
          type: 'webrtc_ice_candidate',
          data: {
            candidate: data.candidate,
            sdpMLineIndex: data.sdpMLineIndex || 0,
            sdpMid: data.sdpMid
          },
          from: websocketService.clientId,
          to: peerId
        });
      }
    });

    peer.on('stream', (stream) => {
      this.remoteStream = stream;
      this.connectionState = 'connected';
      connectionHealthy = true;
      iceErrorCount = 0;
      
      // Stop ICE candidate processing
      this.iceProcessingEnabled = false;
      
      if (this.onRemoteStream) {
        this.onRemoteStream(stream);
      }
    });

    peer.on('connect', () => {
      this.connectionState = 'connected';
      connectionHealthy = true;
      
      // Disable ICE candidate processing once connected
      this.iceProcessingEnabled = false;
      
      if (this.onConnect) {
        this.onConnect(peerId);
      }
    });

    peer.on('error', (error) => {
      
      const errorStr = error.toString();
      
      // ICE candidate errors are usually non-fatal
      if (errorStr.includes('RTCIceCandidate') || errorStr.includes('ICE candidate')) {
        iceErrorCount++;
        
        // Only treat as fatal if we get too many ICE errors without a working stream
        if (iceErrorCount >= maxIceErrors && !connectionHealthy) {
          this.connectionState = 'error';
          this.cleanupPeer(peerId);
          if (this.onError) {
            this.onError(error, peerId);
          }
        } else {
          // Non-fatal ICE error
          this.connectionState = connectionHealthy ? 'connected' : 'connecting';
        }
        return;
      }
      
      // Actual fatal error
      this.connectionState = 'error';
      this.cleanupPeer(peerId);
      
      if (this.onError) {
        this.onError(error, peerId);
      }
    });

    peer.on('close', () => {
    
      // Check if we should attempt reconnection
      if (this.remoteStream && this.remoteStream.active) {
          this.connectionState = 'disconnected';
          
          // Don't cleanup immediately for temporary disconnections
          // The stream might still be working
          setTimeout(() => {
              // Only cleanup if the peer is truly dead and stream is gone
              if (peer.destroyed && (!this.remoteStream || !this.remoteStream.active)) {
                  if (stateInterval) clearInterval(stateInterval);
                  this.cleanupPeer(peerId);
                  
                  if (this.onClose) {
                      this.onClose(peerId);
                  }
              } else {
              }
          }, 15000); // Wait 15 seconds for potential recovery
      } else {
          // No active stream, cleanup immediately
          this.connectionState = 'disconnected';
          if (stateInterval) clearInterval(stateInterval);
          this.cleanupPeer(peerId);
          
          if (this.onClose) {
              this.onClose(peerId);
          }
      }
    });

    // Debug connection state monitoring
    stateInterval = setInterval(() => {
      if (peer.destroyed) {
        clearInterval(stateInterval);
        return;
      }
    }, 5000);

    peer.on('iceStateChange', (state) => {
    });
  }

  setupSignalingListeners() {
    if (this.signalListenersSetup) {
      return;
    }


    websocketService.on('webrtc_offer', async (message) => {
      
      if (!this.localStream) {
        return;
      }
      
      
      try {
        const peer = await this.createPeer(false, message.from);
        
        if (peer && !peer.destroyed) {
          const offerData = {
            type: 'offer',
            sdp: message.data.sdp
          };
          peer.signal(offerData);
        } else {
        }
      } catch (error) {
      }
    });

    websocketService.on('webrtc_answer', (message) => {
      const peer = this.peers.get(message.from);
      
      if (peer && !peer.destroyed) {
        try {
          const answerData = message.data;
          
          if (answerData && answerData.sdp && !answerData.type) {
            answerData.type = 'answer';
          }
          
          peer.signal(answerData);
        } catch (error) {
          this.cleanupPeer(message.from);
        }
      } else {
      }
    });

    websocketService.on('webrtc_ice_candidate', (message) => {
      if (!this.iceProcessingEnabled) {
        return;
      }
      
      const peer = this.peers.get(message.from);
      
      if (peer && !peer.destroyed && !peer.connected) {
          try {
              const candidateData = message.data;
              if (candidateData && candidateData.candidate) {
                  // Ensure proper format for SimplePeer
                  const formattedCandidate = {
                      candidate: candidateData.candidate,
                      sdpMLineIndex: candidateData.sdpMLineIndex,
                      sdpMid: candidateData.sdpMid
                  };
                  peer.signal(formattedCandidate);
              }
          } catch (error) {
          }
      }
    });

    this.signalListenersSetup = true;
  }

  async startBroadcast() {
    return null;
  }

  async joinBroadcast(sellerClientId) {
    
    try {
      const peer = await this.createPeer(true, sellerClientId);
      
      return peer;
    } catch (error) {
      throw error;
    }
  }

  destroy() {
    
    if (this.statsInterval) {
      clearInterval(this.statsInterval);
    }

    this.peers.forEach((peer, clientId) => {
      try {
        peer.destroy();
      } catch (error) {
      }
    });
    this.peers.clear();
    
    if (this.localStream) {
      this.localStream.getTracks().forEach(track => {
        track.stop();
      });
      this.localStream = null;
    }
    
    this.remoteStream = null;
    this.connectionState = 'disconnected';
    this.signalListenersSetup = false;
    this.iceProcessingEnabled = true; // Reset for next connection
    
  }

  getConnectionState() {
    return this.connectionState;
  }

  getActiveConnections() {
    return this.peers.size;
  }

  // Event callbacks
  onRemoteStream = null;
  onConnect = null;
  onError = null;
  onClose = null;
  onScreenShareEnded = null;
}

export default new WebRTCService();