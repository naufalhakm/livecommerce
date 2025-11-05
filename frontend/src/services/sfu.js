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
  }

  async connect(roomId, role = 'viewer') {
    const sfuUrl = import.meta.env.VITE_SFU_URL || 'ws://localhost:8188';
    this.roomId = roomId;
    this.role = role;
    
    return new Promise((resolve, reject) => {
      try {
        // Remove /ws suffix since nginx proxy handles the path
        this.ws = new WebSocket(sfuUrl);
        
        const timeout = setTimeout(() => {
          reject(new Error('SFU connection timeout'));
        }, 5000);
        
        this.ws.onopen = () => {
          clearTimeout(timeout);
          this.joinRoom();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            if (!event.data || event.data.trim() === '') {
              return;
            }
            const message = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
          }
        };

        this.ws.onerror = (error) => {
          clearTimeout(timeout);
          if (this.onError) this.onError(error);
          reject(error);
        };

      } catch (error) {
        if (this.onError) this.onError(error);
        reject(error);
      }
    });
  }

  async joinRoom() {
    this.clientId = `${this.role}_${Date.now()}`;
    
    // Join room first
    this.sendMessage({
      type: 'join',
      data: {
        client_id: this.clientId,
        role: this.role
      },
      room: this.roomId,
      role: this.role
    });
  }

  async setupPeerConnection() {
    this.pc = new RTCPeerConnection({
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' }
      ],
      iceCandidatePoolSize: 10
    });

    this.pc.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendMessage({
          type: 'ice',
          data: {
            candidate: event.candidate.candidate,
            sdpMLineIndex: event.candidate.sdpMLineIndex,
            sdpMid: event.candidate.sdpMid
          },
          room: this.roomId,
          role: this.role,
          client_id: this.clientId
        });
      }
    };

    this.pc.ontrack = (event) => {
      if (this.onRemoteStream && event.streams && event.streams[0]) {
        this.onRemoteStream(event.streams[0]);
      }
    };

    this.pc.onconnectionstatechange = () => {
      if (this.pc.connectionState === 'connected' && this.onConnect) {
        this.onConnect();
      }
    };

    if (this.role === 'publisher' && this.localStream) {
      this.localStream.getTracks().forEach(track => {
        this.pc.addTrack(track, this.localStream);
      });
    }

    const offer = await this.pc.createOffer({
      offerToReceiveAudio: true,
      offerToReceiveVideo: true
    });
    await this.pc.setLocalDescription(offer);

    this.sendMessage({
      type: 'offer',
      data: { sdp: offer.sdp },
      room: this.roomId,
      role: this.role,
      client_id: this.clientId
    });
  }

  async handleMessage(message) {
    try {
      switch (message.type) {
        case 'joined':
          await this.setupPeerConnection();
          break;

        case 'offer':
          // Handle offer from server (for viewers when new track is added)
          if (this.pc && this.role === 'viewer') {
            const offer = new RTCSessionDescription({
              type: 'offer',
              sdp: message.data.sdp
            });
            await this.pc.setRemoteDescription(offer);
            
            const answer = await this.pc.createAnswer();
            await this.pc.setLocalDescription(answer);
            
            this.sendMessage({
              type: 'answer',
              data: { sdp: answer.sdp },
              room: this.roomId,
              role: this.role,
              client_id: this.clientId
            });
          }
          break;

        case 'answer':
          if (this.pc && this.pc.signalingState === 'have-local-offer') {
            const answer = new RTCSessionDescription({
              type: 'answer',
              sdp: message.data.sdp
            });
            await this.pc.setRemoteDescription(answer);
          } else {
          }
          break;

        case 'ice':
          if (message.data.candidate && this.pc && this.pc.remoteDescription) {
            await this.pc.addIceCandidate(new RTCIceCandidate(message.data));
          }
          break;

        default:
      }
    } catch (error) {
    }
  }

  sendMessage(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  async setLocalStream(stream) {
    this.localStream = stream;
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
  }
}

export default new SFUService();