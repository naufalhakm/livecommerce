package services

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type Hub struct {
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	id     string
	roomID string
}

type Message struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	From    string      `json:"from,omitempty"`
	To      string      `json:"to,omitempty"`
	RoomID  string      `json:"room_id,omitempty"`
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if h.rooms[client.roomID] == nil {
				h.rooms[client.roomID] = make(map[*Client]bool)
			}
			h.rooms[client.roomID][client] = true
			h.mutex.Unlock()
			log.Printf("Client %s joined room %s", client.id, client.roomID)

			// Notify room about new client
			h.BroadcastToRoom(client.roomID, Message{
				Type: "user_joined",
				Data: map[string]string{"client_id": client.id},
			})

		case client := <-h.unregister:
			h.mutex.Lock()
			if room, ok := h.rooms[client.roomID]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.send)
					if len(room) == 0 {
						delete(h.rooms, client.roomID)
					}
					log.Printf("Client %s left room %s", client.id, client.roomID)
				}
			}
			h.mutex.Unlock()

			// Notify room about client leaving
			h.BroadcastToRoom(client.roomID, Message{
				Type: "user_left",
				Data: map[string]string{"client_id": client.id},
			})
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if room, ok := h.rooms[roomID]; ok {
		for client := range room {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(room, client)
			}
		}
	}
}

func (h *Hub) SendToClient(roomID, clientID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if room, ok := h.rooms[roomID]; ok {
		for client := range room {
			if client.id == clientID {
				select {
				case client.send <- data:
				default:
					close(client.send)
					delete(room, client)
				}
				break
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		msg.From = c.id
		msg.RoomID = c.roomID

		// Handle different message types
		switch msg.Type {
		case "webrtc_offer", "webrtc_answer", "webrtc_ice_candidate":
			log.Printf("WebRTC signaling: %s from %s to %s in room %s", msg.Type, msg.From, msg.To, c.roomID)
			// Relay WebRTC signaling to specific client or broadcast to room
			if msg.To != "" {
				log.Printf("Sending %s to specific client %s", msg.Type, msg.To)
				c.hub.SendToClient(c.roomID, msg.To, msg)
			} else {
				log.Printf("Broadcasting %s to room %s", msg.Type, c.roomID)
				c.hub.BroadcastToRoom(c.roomID, msg)
			}
		case "chat":
			// Broadcast chat messages to room
			c.hub.BroadcastToRoom(c.roomID, msg)
		default:
			// Broadcast other messages to room
			c.hub.BroadcastToRoom(c.roomID, msg)
		}
	}
}

func (c *Client) writePump() {
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

func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	roomID := r.URL.Query().Get("room_id")

	if clientID == "" || roomID == "" {
		log.Printf("Missing client_id or room_id")
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client_id and room_id are required"))
		conn.Close()
		return
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		id:     clientID,
		roomID: roomID,
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}