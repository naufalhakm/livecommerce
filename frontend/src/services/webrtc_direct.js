class WebRTCDirectService {
  constructor() {
    this.pc = null;
    this.ws = null;
    this.localStream = null;
    this.onRemoteStream = null;
    this.onConnect = null;
    this.onError = null;
    this.clientId = null;
    this.roomId = null;
    this.role = null;
  }

  async connect(roomId, role, clientId) {
    const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080';
    this.roomId = roomId;
    this.role = role;
    this.clientId = clientId || `${role}_${Date.now()}`;
    
    try {
      this.ws = new WebSocket(`${wsUrl}/ws/webrtc?client_id=${this.clientId}&room_id=${this.roomId}&role=${this.role}`);
      
      this.ws.onopen = () => {
        console.log('🔗 WebRTC signaling connected');
        this.setupPeerConnection();
      };

      this.ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        this.handleSignalingMessage(message);
      };

      this.ws.onerror = (error) => {
        console.error('❌ WebRTC signaling error:', error);
        if (this.onError) this.onError(error);
      };

    } catch (error) {
      console.error('❌ WebRTC connection error:', error);
      if (this.onError) this.onError(error);
    }
  }

  setupPeerConnection() {
    this.pc = new RTCPeerConnection({
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' }
      ]
    });

    this.pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendSignalingMessage({
          type: 'webrtc_ice_candidate',
          data: event.candidate.toJSON()
        });
      }
    };

    this.pc.ontrack = (event) => {
      console.log('🎥 Remote stream received');
      if (this.onRemoteStream) {
        this.onRemoteStream(event.streams[0]);
      }
    };

    this.pc.onconnectionstatechange = () => {
      console.log('WebRTC connection state:', this.pc.connectionState);
      if (this.pc.connectionState === 'connected' && this.onConnect) {
        this.onConnect();
      }
    };

    // If publisher, add local stream and create offer
    if (this.role === 'publisher' && this.localStream) {
      this.localStream.getTracks().forEach(track => {
        this.pc.addTrack(track, this.localStream);
      });
    }

    // If viewer, create offer to request stream
    if (this.role === 'viewer') {
      this.createOffer();
    }
  }

  async createOffer() {
    try {
      const offer = await this.pc.createOffer({
        offerToReceiveAudio: true,
        offerToReceiveVideo: true
      });
      await this.pc.setLocalDescription(offer);
      
      this.sendSignalingMessage({
        type: 'webrtc_offer',
        data: offer
      });
    } catch (error) {
      console.error('Error creating offer:', error);
    }
  }

  async handleSignalingMessage(message) {
    try {
      switch (message.type) {
        case 'webrtc_offer':
          if (this.role === 'publisher') {
            await this.pc.setRemoteDescription(new RTCSessionDescription(message.data));
            const answer = await this.pc.createAnswer();
            await this.pc.setLocalDescription(answer);
            
            this.sendSignalingMessage({
              type: 'webrtc_answer',
              data: answer,
              to: message.from
            });
          }
          break;

        case 'webrtc_answer':
          if (this.role === 'viewer') {
            await this.pc.setRemoteDescription(new RTCSessionDescription(message.data));
          }
          break;

        case 'webrtc_ice_candidate':
          await this.pc.addIceCandidate(new RTCIceCandidate(message.data));
          break;

        case 'publisher_left':
          if (this.role === 'viewer' && this.onError) {
            this.onError(new Error('Publisher left the room'));
          }
          break;
      }
    } catch (error) {
      console.error('Error handling signaling message:', error);
    }
  }

  sendSignalingMessage(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  async setLocalStream(stream) {
    this.localStream = stream;
    if (this.pc && this.role === 'publisher') {
      stream.getTracks().forEach(track => {
        this.pc.addTrack(track, stream);
      });
    }
  }

  disconnect() {
    if (this.pc) {
      this.pc.close();
      this.pc = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.localStream = null;
    console.log('🔌 WebRTC Direct: Disconnected');
  }
}

export default new WebRTCDirectService();