import websocketService from './websocket';

class WebRTCService {
  constructor() {
    this.peers = new Map();
    this.localStream = null;
    this.remoteStream = null;
    this.signalListenersSetup = false;
    this.SimplePeer = null;
  }

  async loadSimplePeer() {
    if (!this.SimplePeer) {
      if (window.SimplePeer) {
        this.SimplePeer = window.SimplePeer;
      } else {
        throw new Error('SimplePeer not loaded from CDN');
      }
    }
    return this.SimplePeer;
  }

  async initializeCamera() {
    try {
      this.localStream = await navigator.mediaDevices.getUserMedia({
        video: { width: 1280, height: 720 },
        audio: true
      });
      return this.localStream;
    } catch (error) {
      console.error('Error accessing camera:', error);
      throw error;
    }
  }

  async createPeer(isInitiator = false, targetClientId = null) {
    console.log('Creating peer - initiator:', isInitiator, 'target:', targetClientId);
    console.log('Has local stream:', !!this.localStream);
    
    // Always clean up existing peer to avoid state conflicts
    if (this.peers.has(targetClientId)) {
      console.log('Cleaning up existing peer for:', targetClientId);
      const existingPeer = this.peers.get(targetClientId);
      existingPeer.destroy();
      this.peers.delete(targetClientId);
    }

    if (!this.localStream && !isInitiator) {
      console.error('No local stream available for seller to send!');
      return null;
    }

    const SimplePeer = await this.loadSimplePeer();

    const peerConfig = {
      initiator: isInitiator,
      trickle: true,
      config: {
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' },
          { urls: 'stun:stun1.l.google.com:19302' }
        ]
      }
    };
    
    // Viewer should expect to receive streams
    if (isInitiator) {
      peerConfig.offerOptions = {
        offerToReceiveAudio: true,
        offerToReceiveVideo: true
      };
      console.log('Viewer configured to receive audio/video');
    }

    if (this.localStream && !isInitiator) {
      // Only seller (non-initiator) sends stream
      peerConfig.stream = this.localStream;
      console.log('Seller adding local stream to peer config:', this.localStream.getTracks().length, 'tracks');
    } else if (!isInitiator) {
      console.error('Seller has no local stream to send!');
    } else {
      console.log('Viewer peer - no local stream needed');
    }

    let peer;
    try {
      peer = new SimplePeer(peerConfig);
    } catch (error) {
      console.error('Error creating peer:', error);
      throw error;
    }

    if (targetClientId) {
      this.peers.set(targetClientId, peer);
    }

    peer.on('signal', (data) => {
      console.log('Sending signal:', data.type, 'to:', targetClientId);
      if (data.type === 'offer') {
        websocketService.sendOffer(data, targetClientId);
      } else if (data.type === 'answer') {
        websocketService.sendAnswer(data, targetClientId);
      } else if (data.candidate) {
        websocketService.sendIceCandidate(data, targetClientId);
      }
    });

    peer.on('stream', (stream) => {
      console.log('Stream received on peer:', targetClientId, stream);
      this.remoteStream = stream;
      if (this.onRemoteStream) {
        this.onRemoteStream(stream);
      } else {
        console.warn('No onRemoteStream callback set');
      }
    });

    peer.on('connect', () => {
      console.log('WebRTC connection established with:', targetClientId);
      if (this.onConnect) {
        this.onConnect();
      }
    });

    peer.on('error', (error) => {
      console.error('WebRTC error with', targetClientId, ':', error);
      if (this.onError) {
        this.onError(error);
      }
    });

    peer.on('close', () => {
      console.log('Peer closed:', targetClientId);
      this.peers.delete(targetClientId);
    });

    if (!this.signalListenersSetup) {
      this.setupSignalingListeners();
      this.signalListenersSetup = true;
    }

    return peer;
  }

  setupSignalingListeners() {
    websocketService.on('webrtc_offer', async (message) => {
      console.log('Received offer from:', message.from);
      console.log('Current peers:', Array.from(this.peers.keys()));
      console.log('Has local stream:', !!this.localStream);
      
      // Clean up existing peer if exists
      if (this.peers.has(message.from)) {
        const existingPeer = this.peers.get(message.from);
        existingPeer.destroy();
        this.peers.delete(message.from);
        console.log('Cleaned up existing peer for:', message.from);
      }
      
      try {
        const peer = await this.createPeer(false, message.from);
        console.log('Created peer for:', message.from, peer);
        if (peer && !peer.destroyed) {
          peer.signal(message.data);
          console.log('Signaled offer to peer');
        }
      } catch (error) {
        console.error('Error handling offer:', error);
      }
    });

    websocketService.on('webrtc_answer', (message) => {
      console.log('Received answer from:', message.from);
      const peer = this.peers.get(message.from);
      if (peer && !peer.destroyed) {
        try {
          // Check peer connection state before signaling
          if (peer._pc && peer._pc.signalingState === 'have-local-offer') {
            peer.signal(message.data);
          } else {
            console.warn('Peer not in correct state for answer:', peer._pc?.signalingState);
            // Clean up and ignore this answer
            this.peers.delete(message.from);
          }
        } catch (error) {
          console.error('Error signaling answer:', error);
          this.peers.delete(message.from);
        }
      }
    });

    websocketService.on('webrtc_ice_candidate', (message) => {
      console.log('Received ICE candidate from:', message.from);
      const peer = this.peers.get(message.from);
      if (peer) {
        peer.signal(message.data);
      }
    });
  }

  async startBroadcast() {
    return await this.createPeer(true);
  }

  async joinBroadcast(sellerClientId) {
    return await this.createPeer(true, sellerClientId); // Viewer is initiator
  }

  destroy() {
    this.peers.forEach((peer, clientId) => {
      console.log('Destroying peer for:', clientId);
      peer.destroy();
    });
    this.peers.clear();
    
    if (this.localStream) {
      this.localStream.getTracks().forEach(track => track.stop());
      this.localStream = null;
    }
    
    this.remoteStream = null;
  }

  onRemoteStream = null;
  onConnect = null;
  onError = null;
}

export default new WebRTCService();
