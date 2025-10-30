class SFUService {
  constructor() {
    this.pc = null;
    this.ws = null;
    this.localStream = null;
    this.onRemoteStream = null;
    this.onConnect = null;
    this.onError = null;
  }

  async connect(roomId, role = 'viewer') {
    const sfuUrl = import.meta.env.VITE_SFU_URL || 'ws://localhost:8188';
    
    try {
      this.ws = new WebSocket(sfuUrl);
      
      this.ws.onopen = () => {
        console.log('🔗 Janus WebSocket connected');
        this.joinRoom(roomId, role);
      };

      this.ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        this.handleMessage(message);
      };

      this.ws.onerror = (error) => {
        console.error('❌ Janus WebSocket error:', error);
        if (this.onError) this.onError(error);
      };

    } catch (error) {
      console.error('❌ Janus connection error:', error);
      if (this.onError) this.onError(error);
    }
  }

  async joinRoom(roomId, role) {
    this.pc = new RTCPeerConnection({
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' }
      ]
    });

    this.pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendMessage({
          method: 'trickle',
          params: {
            candidate: event.candidate
          }
        });
      }
    };

    this.pc.ontrack = (event) => {
      console.log('🎥 Janus: Remote stream received');
      if (this.onRemoteStream) {
        this.onRemoteStream(event.streams[0]);
      }
    };

    if (role === 'publisher') {
      // Seller publishes stream
      if (this.localStream) {
        this.localStream.getTracks().forEach(track => {
          this.pc.addTrack(track, this.localStream);
        });
      }

      const offer = await this.pc.createOffer();
      await this.pc.setLocalDescription(offer);

      this.sendMessage({
        method: 'join',
        params: {
          sid: roomId,
          offer: offer,
          config: { codec: 'vp8' }
        }
      });
    } else {
      // Viewer subscribes to stream
      this.sendMessage({
        method: 'join',
        params: {
          sid: roomId
        }
      });
    }
  }

  async handleMessage(message) {
    switch (message.method) {
      case 'offer':
        await this.pc.setRemoteDescription(message.params);
        const answer = await this.pc.createAnswer();
        await this.pc.setLocalDescription(answer);
        
        this.sendMessage({
          method: 'answer',
          params: answer
        });
        break;

      case 'answer':
        await this.pc.setRemoteDescription(message.params);
        break;

      case 'trickle':
        if (message.params.candidate) {
          await this.pc.addIceCandidate(message.params.candidate);
        }
        break;
    }
  }

  sendMessage(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  async setLocalStream(stream) {
    this.localStream = stream;
    console.log('🎥 Janus: Local stream set');
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
    console.log('🔌 Janus: Disconnected');
  }
}

export default new SFUService();