package entities

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type WebRTCMessage struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	Room     string      `json:"room"`
	Role     string      `json:"role"`
	ClientID string      `json:"client_id,omitempty"`
	To       string      `json:"to,omitempty"`
	From     string      `json:"from,omitempty"`
}

type Client struct {
	ID            string
	Conn          *websocket.Conn
	PeerConnection *webrtc.PeerConnection
	Role          string
	RoomID        string
	LocalTracks   []*webrtc.TrackLocalStaticRTP
	ConnectedAt   time.Time
}

type Room struct {
	ID      string
	Clients map[string]*Client
	Mutex   sync.RWMutex
}

type WebRTCConfig struct {
	ICEServers   []webrtc.ICEServer
	SDPSemantics webrtc.SDPSemantics
}