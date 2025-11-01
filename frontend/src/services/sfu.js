class SFUService {
  constructor() {
    this.pc = null;
    this.ws = null;
    this.localStream = null;
    this.onRemoteStream = null;
    this.onConnect = null;
    this.onError = null;
    this.roomId = null;
    this.role = null;
    this.clientId = null;
  }

  async connect(roomId, role = 'viewer') {
    const sfuUrl = import.meta.env.VITE_SFU_URL || 'ws://localhost:8188';
    this.roomId = roomId;
    this.role = role;
    this.clientId = `${role}_${Date.now()}`;
    
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(sfuUrl);
        
        const timeout = setTimeout(() => {
          reject(new Error('SFU connection timeout'));
        }, 10000);
        
        this.ws.onopen = () => {
          clearTimeout(timeout);
          console.log('🔗 SFU WebSocket connected');
          this.joinRoom();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            console.error('❌ SFU: JSON parse error:', error);
          }
        };

        this.ws.onerror = (error) => {
          clearTimeout(timeout);
          console.error('❌ SFU WebSocket error:', error);
          if (this.onError) this.onError(error);
          reject(error);
        };

        this.ws.onclose = () => {
          console.log('🔌 SFU WebSocket closed');
        };

      } catch (error) {
        console.error('❌ SFU connection error:', error);
        if (this.onError) this.onError(error);
        reject(error);
      }
    });
  }

  joinRoom() {
    this.sendMessage({
      type: 'join',
      data: {
        client_id: this.clientId,
        role: this.role
      },
      room: this.roomId
    });
  }

  async handleMessage(message) {
    try {
      switch (message.type) {
        case 'joined':
          console.log('✅ SFU: Joined room, setting up peer connection');
          await this.setupPeerConnection();
          break;

        case 'answer':
          if (this.pc && this.pc.signalingState === 'have-local-offer') {
            const answer = new RTCSessionDescription({
              type: 'answer',
              sdp: message.data.sdp
            });
            await this.pc.setRemoteDescription(answer);
            console.log('✅ SFU: Answer received');
          }
          break;

        case 'ice-candidate':
          if (this.pc && this.pc.remoteDescription) {
            await this.pc.addIceCandidate(new RTCIceCandidate(message.data));
          }
          break;

        default:
          console.log('SFU: Unknown message type:', message.type);
      }
    } catch (error) {
      console.error('❌ SFU: Error handling message:', error);
    }
  }

  async setupPeerConnection() {
    this.pc = new RTCPeerConnection({
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' }
      ]
    });

    this.pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendMessage({
          type: 'ice-candidate',
          data: event.candidate.toJSON(),
          room: this.roomId
        });
      }
    };

    this.pc.ontrack = (event) => {
      console.log('🎥 SFU: Remote stream received');
      if (this.onRemoteStream && event.streams && event.streams[0]) {
        this.onRemoteStream(event.streams[0]);
      }
    };

    this.pc.onconnectionstatechange = () => {
      console.log('SFU connection state:', this.pc.connectionState);
      if (this.pc.connectionState === 'connected' && this.onConnect) {
        this.onConnect();
      } else if (this.pc.connectionState === 'failed') {
        console.error('❌ SFU: Connection failed');
        if (this.onError) this.onError(new Error('WebRTC connection failed'));
      }
    };

    // Add local stream tracks if publisher
    if (this.role === 'publisher' && this.localStream) {
      console.log('🎥 SFU: Adding tracks to peer connection...');
      this.localStream.getTracks().forEach(track => {
        console.log(`🎥 SFU: Adding ${track.kind} track`);
        this.pc.addTrack(track, this.localStream);
      });
      console.log(`🎥 SFU: Added ${this.localStream.getTracks().length} tracks`);
    }

    // Create and send offer
    const offer = await this.pc.createOffer({
      offerToReceiveAudio: true,
      offerToReceiveVideo: true
    });
    await this.pc.setLocalDescription(offer);

    this.sendMessage({
      type: 'offer',
      data: { sdp: offer.sdp },
      room: this.roomId
    });
  }

  sendMessage(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  async setLocalStream(stream) {
    this.localStream = stream;
    console.log('🎥 SFU: Local stream set');
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
    console.log('🔌 SFU: Disconnected');
  }
}

export default new SFUService();