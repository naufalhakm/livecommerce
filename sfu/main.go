package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type SFU struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

type Room struct {
	id        string
	clients   map[string]*Client
	mutex     sync.RWMutex
}

type Client struct {
	id         string
	conn       *websocket.Conn
	pc         *webrtc.PeerConnection
	role       string
	localTracks []*webrtc.TrackLocalStaticRTP
}

type Message struct {
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	Room     string      `json:"room"`
	Role     string      `json:"role"`
	ClientID string      `json:"client_id,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var sfu = &SFU{
	rooms: make(map[string]*Room),
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", handleWebSocket) // Handle root path for nginx proxy
	log.Println("SFU Server starting on :8188")
	log.Fatal(http.ListenAndServe(":8188", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers for production
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		switch msg.Type {
		case "join":
			handleJoin(conn, msg)
		case "offer":
			handleOffer(conn, msg)
		case "ice":
			handleICE(conn, msg)
		}
	}
}

func handleJoin(conn *websocket.Conn, msg Message) {
	data := msg.Data.(map[string]interface{})
	roomID := msg.Room
	clientID := data["client_id"].(string)
	role := data["role"].(string)

	log.Printf("Client %s (%s) joining room %s at %v", clientID, role, roomID, time.Now())
	
	// Log room stats
	sfu.mutex.RLock()
	if existingRoom, exists := sfu.rooms[roomID]; exists {
		existingRoom.mutex.RLock()
		log.Printf("Room %s has %d existing clients", roomID, len(existingRoom.clients))
		existingRoom.mutex.RUnlock()
	}
	sfu.mutex.RUnlock()

	sfu.mutex.Lock()
	room, exists := sfu.rooms[roomID]
	if !exists {
		room = &Room{
			id:      roomID,
			clients: make(map[string]*Client),
		}
		sfu.rooms[roomID] = room
	}
	sfu.mutex.Unlock()

	// Get server IP from environment
	serverIP := os.Getenv("SERVER_IP")
	if serverIP == "" {
		serverIP = "localhost"
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun1.l.google.com:19302"}},
			// Add TURN server for production
			// {URLs: []string{"turn:" + serverIP + ":3478"}, Username: "user", Credential: "pass"},
		},
		BundlePolicy: webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy: webrtc.RTCPMuxPolicyRequire,
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("Failed to create peer connection: %v", err)
		return
	}

	client := &Client{
		id:   clientID,
		conn: conn,
		pc:   pc,
		role: role,
	}

	room.mutex.Lock()
	room.clients[clientID] = client
	room.mutex.Unlock()

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			response := Message{
				Type: "ice",
				Data: candidate.ToJSON(),
				Room: roomID,
				ClientID: clientID,
			}
			if err := conn.WriteJSON(response); err != nil {
				log.Printf("Error sending ICE candidate: %v", err)
			}
		}
	})

	if role == "publisher" {
		pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			log.Printf("Publisher track received: %s", track.Kind())

			// Create local track for forwarding
			localTrack, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, track.ID(), track.StreamID())
			if err != nil {
				log.Printf("Error creating local track: %v", err)
				return
			}

			client.localTracks = append(client.localTracks, localTrack)

			// Add track to all viewers in the room
			room.mutex.RLock()
			for _, otherClient := range room.clients {
				if otherClient.role == "viewer" && otherClient.id != clientID {
					if _, err := otherClient.pc.AddTrack(localTrack); err != nil {
						log.Printf("Error adding track to viewer %s: %v", otherClient.id, err)
					} else {
						log.Printf("Track added to viewer %s", otherClient.id)
					}
				}
			}
			room.mutex.RUnlock()

			// Forward RTP packets with optimized buffer
			go func() {
				rtpBuf := make([]byte, 1500) // Slightly larger buffer
				for {
					i, _, readErr := track.Read(rtpBuf)
					if readErr != nil {
						return
					}
					if _, writeErr := localTrack.Write(rtpBuf[:i]); writeErr != nil {
						return
					}
				}
			}()
		})
	} else {
		// Viewer - add existing tracks from all publishers
		room.mutex.RLock()
		for _, otherClient := range room.clients {
			if otherClient.role == "publisher" {
				for _, track := range otherClient.localTracks {
					if _, err := pc.AddTrack(track); err != nil {
						log.Printf("Error adding existing track to viewer %s: %v", clientID, err)
					} else {
						log.Printf("Existing track added to viewer %s", clientID)
					}
				}
			}
		}
		room.mutex.RUnlock()
	}

	// Send joined response
	response := Message{
		Type: "joined",
		Data: map[string]string{"status": "success"},
		Room: roomID,
		ClientID: clientID,
	}
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("Error sending joined response: %v", err)
	}
}

func handleOffer(conn *websocket.Conn, msg Message) {
	data := msg.Data.(map[string]interface{})
	roomID := msg.Room
	clientID := msg.ClientID

	// If no clientID in message, find by connection
	if clientID == "" {
		sfu.mutex.RLock()
		room := sfu.rooms[roomID]
		sfu.mutex.RUnlock()
		
		if room != nil {
			room.mutex.RLock()
			for _, client := range room.clients {
				if client.conn == conn {
					clientID = client.id
					break
				}
			}
			room.mutex.RUnlock()
		}
	}

	sfu.mutex.RLock()
	room := sfu.rooms[roomID]
	sfu.mutex.RUnlock()

	if room == nil {
		return
	}

	room.mutex.RLock()
	client := room.clients[clientID]
	room.mutex.RUnlock()

	if client == nil {
		return
	}

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  data["sdp"].(string),
	}

	if err := client.pc.SetRemoteDescription(offer); err != nil {
		log.Printf("Error setting remote description: %v", err)
		return
	}

	answer, err := client.pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("Error creating answer: %v", err)
		return
	}

	if err := client.pc.SetLocalDescription(answer); err != nil {
		log.Printf("Error setting local description: %v", err)
		return
	}

	response := Message{
		Type: "answer",
		Data: map[string]string{"sdp": answer.SDP},
		Room: roomID,
		ClientID: clientID,
	}
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("Error sending answer: %v", err)
	}
}

func handleICE(conn *websocket.Conn, msg Message) {
	data := msg.Data.(map[string]interface{})
	roomID := msg.Room
	clientID := msg.ClientID

	// If no clientID in message, find by connection
	if clientID == "" {
		sfu.mutex.RLock()
		room := sfu.rooms[roomID]
		sfu.mutex.RUnlock()
		
		if room != nil {
			room.mutex.RLock()
			for _, client := range room.clients {
				if client.conn == conn {
					clientID = client.id
					break
				}
			}
			room.mutex.RUnlock()
		}
	}

	sfu.mutex.RLock()
	room := sfu.rooms[roomID]
	sfu.mutex.RUnlock()

	if room == nil {
		return
	}

	room.mutex.RLock()
	client := room.clients[clientID]
	room.mutex.RUnlock()

	if client == nil {
		return
	}

	candidate := webrtc.ICECandidateInit{
		Candidate: data["candidate"].(string),
	}

	if sdpMLineIndex, ok := data["sdpMLineIndex"]; ok {
		idx := uint16(sdpMLineIndex.(float64))
		candidate.SDPMLineIndex = &idx
	}

	if sdpMid, ok := data["sdpMid"]; ok {
		mid := sdpMid.(string)
		candidate.SDPMid = &mid
	}

	client.pc.AddICECandidate(candidate)
}