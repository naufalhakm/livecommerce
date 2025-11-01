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

	var currentClient *Client
	var currentRoom *Room

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		switch msg.Type {
		case "join":
			currentClient, currentRoom = handleJoin(conn, msg)
		case "offer":
			handleOffer(conn, msg)
		case "ice":
			handleICE(conn, msg)
		}
	}

	// Clean up client on disconnect
	if currentClient != nil && currentRoom != nil {
		log.Printf("🔌 Client %s disconnecting, cleaning up...", currentClient.id)
		cleanupClient(currentClient, currentRoom)
	}
}

func handleJoin(conn *websocket.Conn, msg Message) (*Client, *Room) {
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
			{URLs: []string{"stun:stun2.l.google.com:19302"}},
			{URLs: []string{"stun:stun3.l.google.com:19302"}},
			{URLs: []string{"stun:stun4.l.google.com:19302"}},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("Failed to create peer connection: %v", err)
		return nil, nil
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
			// Forward ICE candidate to all other clients in room
			room.mutex.RLock()
			for _, otherClient := range room.clients {
				if otherClient.id != clientID {
					response := Message{
						Type: "ice",
						Data: candidate.ToJSON(),
						Room: roomID,
						ClientID: clientID,
					}
					// Check if connection is still open before sending
					if isConnectionOpen(otherClient.conn) {
						if err := otherClient.conn.WriteJSON(response); err != nil {
							log.Printf("Error sending ICE candidate to %s: %v", otherClient.id, err)
						}
					}
				}
			}
			room.mutex.RUnlock()
		}
	})

	if role == "publisher" {
		pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			log.Printf("🎥 Publisher track received: %s from %s", track.Kind(), clientID)

			// Create local track for forwarding
			localTrack, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, track.ID(), track.StreamID())
			if err != nil {
				log.Printf("Error creating local track: %v", err)
				return
			}

			client.localTracks = append(client.localTracks, localTrack)

			// Add track to all viewers and trigger renegotiation
			room.mutex.RLock()
			for _, otherClient := range room.clients {
				if otherClient.role == "viewer" && otherClient.id != clientID {
					if _, err := otherClient.pc.AddTrack(localTrack); err != nil {
						log.Printf("❌ Error adding track to viewer %s: %v", otherClient.id, err)
					} else {
						log.Printf("✅ Track added to viewer %s, triggering renegotiation", otherClient.id)
						
						// Create new offer for viewer
						go func(viewerClient *Client) {
							offer, err := viewerClient.pc.CreateOffer(nil)
							if err != nil {
								log.Printf("Error creating offer for viewer %s: %v", viewerClient.id, err)
								return
							}
							
							if err := viewerClient.pc.SetLocalDescription(offer); err != nil {
								log.Printf("Error setting local description for viewer %s: %v", viewerClient.id, err)
								return
							}
							
							response := Message{
								Type: "offer",
								Data: map[string]string{"sdp": offer.SDP},
								Room: roomID,
								ClientID: viewerClient.id,
							}
							if isConnectionOpen(viewerClient.conn) {
								viewerClient.conn.WriteJSON(response)
							}
						}(otherClient)
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

	return client, room
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
		log.Printf("❌ Error setting remote description for %s: %v", clientID, err)
		return
	}
	log.Printf("✅ Remote description set for %s", clientID)

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
	if isConnectionOpen(conn) {
		if err := conn.WriteJSON(response); err != nil {
			log.Printf("Error sending answer: %v", err)
		}
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

func cleanupClient(client *Client, room *Room) {
	room.mutex.Lock()
	delete(room.clients, client.id)
	room.mutex.Unlock()

	if client.pc != nil {
		client.pc.Close()
	}

	log.Printf("✅ Client %s cleaned up from room %s", client.id, room.id)
}

func isConnectionOpen(conn *websocket.Conn) bool {
	// Check if connection is nil first
	if conn == nil {
		return false
	}
	
	// Try to write a ping to check if connection is alive
	err := conn.WriteMessage(websocket.PingMessage, []byte{})
	return err == nil
}

// Periodic cleanup of stale connections
func init() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			cleanupStaleConnections()
		}
	}()
}

func cleanupStaleConnections() {
	sfu.mutex.RLock()
	for roomID, room := range sfu.rooms {
		room.mutex.Lock()
		for clientID, client := range room.clients {
			if !isConnectionOpen(client.conn) {
				log.Printf("🧹 Cleaning up stale client %s from room %s", clientID, roomID)
				delete(room.clients, clientID)
				if client.pc != nil {
					client.pc.Close()
				}
			}
		}
		room.mutex.Unlock()
	}
	sfu.mutex.RUnlock()
}