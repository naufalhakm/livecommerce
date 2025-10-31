package services

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type WebRTCSignalingHub struct {
	rooms      map[string]*SignalingRoom
	register   chan *SignalingClient
	unregister chan *SignalingClient
	mutex      sync.RWMutex
}

type SignalingRoom struct {
	id        string
	publisher *SignalingClient
	viewers   map[string]*SignalingClient
	mutex     sync.RWMutex
}

type SignalingClient struct {
	hub    *WebRTCSignalingHub
	conn   *websocket.Conn
	send   chan []byte
	id     string
	roomID string
	role   string // "publisher" or "viewer"
}

type SignalingMessage struct {
	Type   string      `json:"type"`
	Data   interface{} `json:"data"`
	From   string      `json:"from,omitempty"`
	To     string      `json:"to,omitempty"`
	RoomID string      `json:"room_id,omitempty"`
}

func NewWebRTCSignalingHub() *WebRTCSignalingHub {
	return &WebRTCSignalingHub{
		rooms:      make(map[string]*SignalingRoom),
		register:   make(chan *SignalingClient),
		unregister: make(chan *SignalingClient),
	}
}

func (h *WebRTCSignalingHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			room, exists := h.rooms[client.roomID]
			if !exists {
				room = &SignalingRoom{
					id:      client.roomID,
					viewers: make(map[string]*SignalingClient),
				}
				h.rooms[client.roomID] = room
			}

			if client.role == "publisher" {
				room.publisher = client
				log.Printf("Publisher joined room %s", client.roomID)
			} else {
				room.viewers[client.id] = client
				log.Printf("Viewer %s joined room %s", client.id, client.roomID)
			}
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if room, ok := h.rooms[client.roomID]; ok {
				if client.role == "publisher" {
					room.publisher = nil
					// Notify all viewers that publisher left
					for _, viewer := range room.viewers {
						select {
						case viewer.send <- []byte(`{"type":"publisher_left"}`):
						default:
							close(viewer.send)
							delete(room.viewers, viewer.id)
						}
					}
				} else {
					delete(room.viewers, client.id)
				}
				close(client.send)
			}
			h.mutex.Unlock()
		}
	}
}

func (h *WebRTCSignalingHub) HandleWebRTCSignaling(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebRTC signaling upgrade error: %v", err)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	roomID := r.URL.Query().Get("room_id")
	role := r.URL.Query().Get("role")

	if clientID == "" || roomID == "" || role == "" {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client_id, room_id, and role are required"))
		conn.Close()
		return
	}

	client := &SignalingClient{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		id:     clientID,
		roomID: roomID,
		role:   role,
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *SignalingClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg SignalingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		msg.From = c.id
		msg.RoomID = c.roomID

		// Route WebRTC signaling messages
		c.hub.mutex.RLock()
		room := c.hub.rooms[c.roomID]
		c.hub.mutex.RUnlock()

		if room == nil {
			continue
		}

		switch msg.Type {
		case "webrtc_offer":
			// Forward offer from viewer to publisher
			if room.publisher != nil {
				data, _ := json.Marshal(msg)
				select {
				case room.publisher.send <- data:
				default:
					close(room.publisher.send)
				}
			}

		case "webrtc_answer":
			// Forward answer from publisher to specific viewer
			if msg.To != "" {
				if viewer, ok := room.viewers[msg.To]; ok {
					data, _ := json.Marshal(msg)
					select {
					case viewer.send <- data:
					default:
						close(viewer.send)
						delete(room.viewers, msg.To)
					}
				}
			}

		case "webrtc_ice_candidate":
			// Forward ICE candidates between peers
			if c.role == "publisher" && msg.To != "" {
				if viewer, ok := room.viewers[msg.To]; ok {
					data, _ := json.Marshal(msg)
					select {
					case viewer.send <- data:
					default:
						close(viewer.send)
						delete(room.viewers, msg.To)
					}
				}
			} else if c.role == "viewer" && room.publisher != nil {
				data, _ := json.Marshal(msg)
				select {
				case room.publisher.send <- data:
				default:
					close(room.publisher.send)
				}
			}
		}
	}
}

func (c *SignalingClient) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}