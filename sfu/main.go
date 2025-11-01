package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type Hub struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

type Room struct {
	id        string
	clients   map[string]*Client
	mutex     sync.RWMutex
}

type Client struct {
	id       string
	conn     *websocket.Conn
	pc       *webrtc.PeerConnection
	role     string
	tracks   []*webrtc.TrackLocalStaticRTP
}

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Room string          `json:"room"`
	From string          `json:"from,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var hub = &Hub{
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
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		switch msg.Type {
		case "join":
			handleJoin(conn, msg)
		case "offer":
			handleOffer(conn, msg)
		case "answer":
			handleAnswer(conn, msg)
		case "ice-candidate":
			handleIceCandidate(conn, msg)
		}
	}
}

func handleJoin(conn *websocket.Conn, msg Message) {
	var joinData struct {
		ClientID string `json:"client_id"`
		Role     string `json:"role"`
	}
	json.Unmarshal(msg.Data, &joinData)

	roomID := msg.Room
	log.Printf("Client %s (%s) joining room %s", joinData.ClientID, joinData.Role, roomID)

	// Get or create room
	hub.mutex.Lock()
	room, exists := hub.rooms[roomID]
	if !exists {
		room = &Room{
			id:      roomID,
			clients: make(map[string]*Client),
		}
		hub.rooms[roomID] = room
	}
	hub.mutex.Unlock()

	// Create WebRTC peer connection
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("Failed to create peer connection: %v", err)
		return
	}

	client := &Client{
		id:   joinData.ClientID,
		conn: conn,
		pc:   pc,
		role: joinData.Role,
	}

	// Add client to room
	room.mutex.Lock()
	room.clients[joinData.ClientID] = client
	room.mutex.Unlock()

	// Setup ICE candidate handling
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			candidateData, _ := json.Marshal(candidate.ToJSON())
			response := Message{
				Type: "ice-candidate",
				Data: candidateData,
				Room: roomID,
				From: joinData.ClientID,
			}
			conn.WriteJSON(response)
		}
	})

	// Setup connection state monitoring
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("Client %s connection state: %s", joinData.ClientID, state.String())
	})

	if joinData.Role == "publisher" {
		// Publisher receives tracks and forwards them
		pc.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			log.Printf("Received track: %s", remoteTrack.Kind().String())

			// Create a local track to forward this track to viewers
			localTrack, err := webrtc.NewTrackLocalStaticRTP(
				remoteTrack.Codec().RTPCodecCapability,
				remoteTrack.ID(),
				remoteTrack.StreamID(),
			)
			if err != nil {
				log.Printf("Failed to create local track: %v", err)
				return
			}

			client.tracks = append(client.tracks, localTrack)

			// Add this track to all viewers in the room
			room.mutex.RLock()
			for _, viewer := range room.clients {
				if viewer.role == "viewer" && viewer.id != joinData.ClientID {
					_, err := viewer.pc.AddTrack(localTrack)
					if err != nil {
						log.Printf("Failed to add track to viewer %s: %v", viewer.id, err)
					} else {
						log.Printf("Added track to viewer %s", viewer.id)
					}
				}
			}
			room.mutex.RUnlock()

			// Start forwarding RTP packets
			go func() {
				rtpBuf := make([]byte, 1400)
				for {
					i, _, readErr := remoteTrack.Read(rtpBuf)
					if readErr != nil {
						return
					}
					_, writeErr := localTrack.Write(rtpBuf[:i])
					if writeErr != nil {
						return
					}
				}
			}()
		})
	} else {
		// Viewer - add existing tracks from publishers
		room.mutex.RLock()
		for _, publisher := range room.clients {
			if publisher.role == "publisher" {
				for _, track := range publisher.tracks {
					_, err := pc.AddTrack(track)
					if err != nil {
						log.Printf("Failed to add existing track to viewer: %v", err)
					} else {
						log.Printf("Added existing track to viewer %s", joinData.ClientID)
					}
				}
			}
		}
		room.mutex.RUnlock()
	}

	// Send join success response
	response := Message{
		Type: "joined",
		Data: json.RawMessage(`{"status":"success"}`),
		Room: roomID,
		From: "server",
	}
	conn.WriteJSON(response)
}

func handleOffer(conn *websocket.Conn, msg Message) {
	var offerData struct {
		SDP string `json:"sdp"`
	}
	json.Unmarshal(msg.Data, &offerData)

	client := findClientByConn(conn, msg.Room)
	if client == nil {
		return
	}

	// Set remote description
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offerData.SDP,
	}

	err := client.pc.SetRemoteDescription(offer)
	if err != nil {
		log.Printf("Failed to set remote description: %v", err)
		return
	}

	// Create answer
	answer, err := client.pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("Failed to create answer: %v", err)
		return
	}

	// Set local description
	err = client.pc.SetLocalDescription(answer)
	if err != nil {
		log.Printf("Failed to set local description: %v", err)
		return
	}

	// Send answer back
	answerData, _ := json.Marshal(map[string]string{"sdp": answer.SDP})
	response := Message{
		Type: "answer",
		Data: answerData,
		Room: msg.Room,
		From: "server",
	}
	conn.WriteJSON(response)
}

func handleAnswer(conn *websocket.Conn, msg Message) {
	var answerData struct {
		SDP string `json:"sdp"`
	}
	json.Unmarshal(msg.Data, &answerData)

	client := findClientByConn(conn, msg.Room)
	if client == nil {
		return
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answerData.SDP,
	}

	err := client.pc.SetRemoteDescription(answer)
	if err != nil {
		log.Printf("Failed to set remote description: %v", err)
	}
}

func handleIceCandidate(conn *websocket.Conn, msg Message) {
	var candidateData webrtc.ICECandidateInit
	json.Unmarshal(msg.Data, &candidateData)

	client := findClientByConn(conn, msg.Room)
	if client == nil {
		return
	}

	err := client.pc.AddICECandidate(candidateData)
	if err != nil {
		log.Printf("Failed to add ICE candidate: %v", err)
	}
}

func findClientByConn(conn *websocket.Conn, roomID string) *Client {
	hub.mutex.RLock()
	room := hub.rooms[roomID]
	hub.mutex.RUnlock()

	if room == nil {
		return nil
	}

	room.mutex.RLock()
	defer room.mutex.RUnlock()

	for _, client := range room.clients {
		if client.conn == conn {
			return client
		}
	}
	return nil
}