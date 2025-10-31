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

  async initializeCamera(facingMode = 'user') {
    try {
      this.localStream = await navigator.mediaDevices.getUserMedia({
        video: { 
          width: { ideal: 1280, max: 1280 }, 
          height: { ideal: 720, max: 720 },
          frameRate: { ideal: 30, max: 30 },
          facingMode: facingMode
        },
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          sampleRate: 44100
        }
      });
      console.log('🎥 Camera stream initialized:', this.localStream.getTracks().length, 'tracks');
      return this.localStream;
    } catch (error) {
      console.error('Error accessing camera:', error);
      throw error;
    }
  }
  
  async initializeScreenShare() {
    try {
      this.localStream = await navigator.mediaDevices.getDisplayMedia({
        video: { mediaSource: 'screen' },
        audio: true
      });
      console.log('🖥️ Screen share initialized:', this.localStream.getTracks().length, 'tracks');
      return this.localStream;
    } catch (error) {
      console.error('Error accessing screen share:', error);
      throw error;
    }
  }

  async createPeer(isInitiator = false, targetClientId = null) {
    console.log('🔧 Creating peer - initiator:', isInitiator, 'target:', targetClientId);
    console.log('Has local stream:', !!this.localStream);
    
    // Generate unique peer ID to avoid conflicts
    const peerId = targetClientId || `peer-${Date.now()}`;
    
    // Always clean up existing peer to avoid state conflicts
    if (this.peers.has(peerId)) {
      console.log('🧹 Cleaning up existing peer for:', peerId);
      const existingPeer = this.peers.get(peerId);
      existingPeer.destroy();
      this.peers.delete(peerId);
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

    if (peerId) {
      this.peers.set(peerId, peer);
    }

    peer.on('signal', (data) => {
      console.log('📡 Sending signal:', data.type, 'to:', targetClientId);
      if (data.type === 'offer') {
        console.log('📤 VIEWER: Sending offer to seller');
        websocketService.sendOffer(data, targetClientId);
      } else if (data.type === 'answer') {
        console.log('📤 SELLER: Sending answer to viewer');
        websocketService.sendAnswer(data, targetClientId);
      } else if (data.candidate) {
        console.log('📤 Sending ICE candidate');
        websocketService.sendIceCandidate(data, targetClientId);
      }
    });

    peer.on('stream', (stream) => {
      console.log('🎥 Stream received on peer:', peerId);
      console.log('Stream details:', {
        id: stream.id,
        active: stream.active,
        videoTracks: stream.getVideoTracks().length,
        audioTracks: stream.getAudioTracks().length
      });
      this.remoteStream = stream;
      if (this.onRemoteStream) {
        console.log('✅ Calling onRemoteStream callback');
        this.onRemoteStream(stream);
      } else {
        console.warn('❌ No onRemoteStream callback set');
      }
    });

    peer.on('connect', () => {
      console.log('✅ WebRTC connection established with:', peerId);
      console.log('📊 Total active connections:', this.peers.size);
      if (this.onConnect) {
        this.onConnect();
      }
    });

    peer.on('error', (error) => {
      console.error('WebRTC error with', peerId, ':', error);
      // Clean up failed peer
      this.peers.delete(peerId);
      if (this.onError) {
        this.onError(error);
      }
    });

    peer.on('close', () => {
      console.log('🔴 Peer closed:', peerId);
      this.peers.delete(peerId);
      console.log('📊 Remaining connections:', this.peers.size);
    });

    // Signaling listeners are set up globally, not per peer
    return peer;
  }

  setupSignalingListeners() {
    console.log('🔧 Setting up WebRTC signaling listeners');
    websocketService.on('webrtc_offer', async (message) => {
      console.log('🔥 SELLER: Received offer from viewer:', message.from);
      console.log('Current peers:', Array.from(this.peers.keys()));
      console.log('Has local stream:', !!this.localStream);
      console.log('Local stream tracks:', this.localStream?.getTracks().length || 0);
      console.log('Local stream active:', this.localStream?.active);
      
      if (!this.localStream) {
        console.error('❌ SELLER: No local stream to send to viewer!');
        console.log('🔍 SELLER: Checking if stream exists in video element...');
        
        // Try to get stream from video element as fallback
        const videoElement = document.querySelector('video');
        console.log('🔍 SELLER: Video element found:', !!videoElement);
        console.log('🔍 SELLER: Video srcObject:', !!videoElement?.srcObject);
        
        if (videoElement && videoElement.srcObject) {
          console.log('✅ SELLER: Found stream in video element, using as fallback');
          this.localStream = videoElement.srcObject;
          console.log('✅ SELLER: Stream tracks from video:', this.localStream.getTracks().length);
        } else {
          console.error('❌ SELLER: No stream found anywhere, cannot proceed');
          return;
        }
      }
      
      // Always clean up existing peer to avoid state conflicts
      if (this.peers.has(message.from)) {
        const existingPeer = this.peers.get(message.from);
        console.log('🧹 SELLER: Cleaning up existing peer for:', message.from);
        existingPeer.destroy();
        this.peers.delete(message.from);
      }
      
      console.log('📊 SELLER: Current active peers:', this.peers.size);
      
      try {
        // Seller responds to viewer's offer (seller is not initiator)
        const peer = await this.createPeer(false, message.from);
        console.log('✅ SELLER: Created peer for viewer:', message.from);
        if (peer && !peer.destroyed) {
          console.log('✅ SELLER: Signaling offer to peer...');
          peer.signal(message.data);
          console.log('✅ SELLER: Offer signaled, peer will generate answer');
        }
      } catch (error) {
        console.error('❌ SELLER: Error handling offer:', error);
      }
    });

    websocketService.on('webrtc_answer', (message) => {
      console.log('🔥 VIEWER: Received answer from seller:', message.from);
      const peer = this.peers.get(message.from);
      if (peer && !peer.destroyed) {
        try {
          // Check peer state before signaling answer
          if (peer._pc && peer._pc.signalingState === 'have-local-offer') {
            console.log('✅ VIEWER: Peer in correct state, signaling answer');
            peer.signal(message.data);
            console.log('✅ VIEWER: Answer signaled, connection should establish');
          } else {
            console.warn('⚠️ VIEWER: Peer not in correct state:', peer._pc?.signalingState);
            // Recreate peer if in wrong state
            this.peers.delete(message.from);
          }
        } catch (error) {
          console.error('❌ VIEWER: Error signaling answer:', error);
          this.peers.delete(message.from);
        }
      } else {
        console.error('❌ VIEWER: No peer found for answer from:', message.from);
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
    console.log('Joining broadcast for seller:', sellerClientId);
    // Viewer initiates connection to seller
    const peer = await this.createPeer(true, sellerClientId);
    console.log('Viewer peer created:', peer);
    return peer;
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
