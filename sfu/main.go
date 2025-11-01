package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type SFU struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

type Room struct {
	id          string
	publisher   *Client
	viewers     map[string]*Client
	localTracks []*webrtc.TrackLocalStaticRTP
	mutex       sync.RWMutex
}

type Client struct {
	id   string
	conn *websocket.Conn
	pc   *webrtc.PeerConnection
	role string
}

type Message struct {
	Type     string          `json:"type"`
	Data     json.RawMessage `json:"data"`
	Room     string          `json:"room"`
	Role     string          `json:"role"`
	ClientID string          `json:"client_id,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var sfu = &SFU{
	rooms: make(map[string]*Room),
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/", handleWebSocket)
	log.Println("SFU Server starting on :8188")
	log.Fatal(http.ListenAndServe(":8188", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
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
		case "answer":
			handleAnswer(conn, msg)
		case "ice":
			handleICE(conn, msg)
		}
	}

	// Cleanup on disconnect
	if currentClient != nil && currentRoom != nil {
		log.Printf("🔌 Client %s disconnecting", currentClient.id)
		cleanupClient(currentClient, currentRoom)
	}
}

func handleJoin(conn *websocket.Conn, msg Message) (*Client, *Room) {
	var data struct {
		ClientID string `json:"client_id"`
		Role     string `json:"role"`
	}
	json.Unmarshal(msg.Data, &data)

	roomID := msg.Room
	log.Printf("Client %s (%s) joining room %s", data.ClientID, data.Role, roomID)

	// Get or create room
	sfu.mutex.Lock()
	room, exists := sfu.rooms[roomID]
	if !exists {
		room = &Room{
			id:      roomID,
			viewers: make(map[string]*Client),
		}
		sfu.rooms[roomID] = room
	}
	sfu.mutex.Unlock()

	// Create peer connection
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
		id:   data.ClientID,
		conn: conn,
		pc:   pc,
		role: data.Role,
	}

	// Setup ICE candidate handling
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			response := Message{
				Type: "ice",
				Data: mustMarshal(candidate.ToJSON()),
				Room: roomID,
			}
			conn.WriteJSON(response)
		}
	})

	room.mutex.Lock()
	if data.Role == "publisher" {
		if room.publisher != nil {
			// Close existing publisher
			room.publisher.pc.Close()
		}
		room.publisher = client
		
		// Setup track handling for publisher
		pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			log.Printf("🎥 Received %s track from publisher", track.Kind())
			
			// Create local track for forwarding
			localTrack, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, track.ID(), track.StreamID())
			if err != nil {
				log.Printf("Error creating local track: %v", err)
				return
			}
			
			// Add to room's local tracks
			room.localTracks = append(room.localTracks, localTrack)
			
			// Add track to all existing viewers
			for _, viewer := range room.viewers {
				if _, err := viewer.pc.AddTrack(localTrack); err != nil {
					log.Printf("Error adding track to viewer: %v", err)
				} else {
					log.Printf("✅ Track forwarded to viewer %s", viewer.id)
					// Trigger renegotiation for viewer
					go createOfferForViewer(viewer, roomID)
				}
			}
			
			// Forward RTP packets
			go func() {
				rtpBuf := make([]byte, 1500)
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
		room.viewers[data.ClientID] = client
		
		// Add existing local tracks to this viewer
		for _, localTrack := range room.localTracks {
			if _, err := pc.AddTrack(localTrack); err != nil {
				log.Printf("Error adding existing track to viewer: %v", err)
			} else {
				log.Printf("✅ Existing track added to viewer %s", data.ClientID)
			}
		}
	}
	room.mutex.Unlock()

	// Send joined response
	response := Message{
		Type: "joined",
		Data: mustMarshal(map[string]string{"status": "success"}),
		Room: roomID,
	}
	conn.WriteJSON(response)

	return client, room
}

func handleOffer(conn *websocket.Conn, msg Message) {
	var data struct {
		SDP string `json:"sdp"`
	}
	json.Unmarshal(msg.Data, &data)

	client := findClientByConn(conn, msg.Room)
	if client == nil {
		return
	}

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  data.SDP,
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
		Data: mustMarshal(map[string]string{"sdp": answer.SDP}),
		Room: msg.Room,
	}
	conn.WriteJSON(response)
}

func handleAnswer(conn *websocket.Conn, msg Message) {
	var data struct {
		SDP string `json:"sdp"`
	}
	json.Unmarshal(msg.Data, &data)

	client := findClientByConn(conn, msg.Room)
	if client == nil {
		return
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  data.SDP,
	}

	if err := client.pc.SetRemoteDescription(answer); err != nil {
		log.Printf("Error setting remote description: %v", err)
		return
	}
}

func handleICE(conn *websocket.Conn, msg Message) {
	var data webrtc.ICECandidateInit
	json.Unmarshal(msg.Data, &data)

	client := findClientByConn(conn, msg.Room)
	if client == nil {
		return
	}

	if err := client.pc.AddICECandidate(data); err != nil {
		log.Printf("Error adding ICE candidate: %v", err)
	}
}

func createOfferForViewer(viewer *Client, roomID string) {
	offer, err := viewer.pc.CreateOffer(nil)
	if err != nil {
		log.Printf("Error creating offer for viewer: %v", err)
		return
	}

	if err := viewer.pc.SetLocalDescription(offer); err != nil {
		log.Printf("Error setting local description for viewer: %v", err)
		return
	}

	response := Message{
		Type: "offer",
		Data: mustMarshal(map[string]string{"sdp": offer.SDP}),
		Room: roomID,
	}
	viewer.conn.WriteJSON(response)
}

func findClientByConn(conn *websocket.Conn, roomID string) *Client {
	sfu.mutex.RLock()
	room := sfu.rooms[roomID]
	sfu.mutex.RUnlock()

	if room == nil {
		return nil
	}

	room.mutex.RLock()
	defer room.mutex.RUnlock()

	if room.publisher != nil && room.publisher.conn == conn {
		return room.publisher
	}

	for _, viewer := range room.viewers {
		if viewer.conn == conn {
			return viewer
		}
	}

	return nil
}

func cleanupClient(client *Client, room *Room) {
	room.mutex.Lock()
	defer room.mutex.Unlock()

	if client.role == "publisher" && room.publisher == client {
		room.publisher = nil
		log.Printf("✅ Publisher %s removed from room %s", client.id, room.id)
	} else {
		delete(room.viewers, client.id)
		log.Printf("✅ Viewer %s removed from room %s", client.id, room.id)
	}

	if client.pc != nil {
		client.pc.Close()
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}