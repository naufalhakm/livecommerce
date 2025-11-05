package webrtc

import (
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/repositories"
	"sync"
)

type memoryWebRTCRepository struct {
	rooms map[string]*entities.Room
	mutex sync.RWMutex
}

func NewMemoryWebRTCRepository() repositories.WebRTCRepository {
	return &memoryWebRTCRepository{
		rooms: make(map[string]*entities.Room),
	}
}

func (r *memoryWebRTCRepository) CreateRoom(roomID string) *entities.Room {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	room := &entities.Room{
		ID:      roomID,
		Clients: make(map[string]*entities.Client),
	}
	r.rooms[roomID] = room
	return room
}

func (r *memoryWebRTCRepository) GetRoom(roomID string) *entities.Room {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.rooms[roomID]
}

func (r *memoryWebRTCRepository) RemoveRoom(roomID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.rooms, roomID)
}

func (r *memoryWebRTCRepository) AddClientToRoom(roomID string, client *entities.Client) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	room, exists := r.rooms[roomID]
	if !exists {
		room = &entities.Room{
			ID:      roomID,
			Clients: make(map[string]*entities.Client),
		}
		r.rooms[roomID] = room
	}

	room.Mutex.Lock()
	room.Clients[client.ID] = client
	room.Mutex.Unlock()
}

func (r *memoryWebRTCRepository) RemoveClientFromRoom(roomID string, clientID string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	room, exists := r.rooms[roomID]
	if !exists {
		return
	}

	room.Mutex.Lock()
	if client, exists := room.Clients[clientID]; exists {
		if client.PeerConnection != nil {
			client.PeerConnection.Close()
		}
		delete(room.Clients, clientID)
	}
	room.Mutex.Unlock()
}

func (r *memoryWebRTCRepository) GetClient(roomID, clientID string) *entities.Client {
	room := r.GetRoom(roomID)
	if room == nil {
		return nil
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()
	return room.Clients[clientID]
}

func (r *memoryWebRTCRepository) GetRoomClients(roomID string) []*entities.Client {
	room := r.GetRoom(roomID)
	if room == nil {
		return nil
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	clients := make([]*entities.Client, 0, len(room.Clients))
	for _, client := range room.Clients {
		clients = append(clients, client)
	}
	return clients
}

func (r *memoryWebRTCRepository) BroadcastToRoom(roomID string, message interface{}, excludeClientID string) error {
	room := r.GetRoom(roomID)
	if room == nil {
		return nil
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	for _, client := range room.Clients {
		if client.ID != excludeClientID && client.Conn != nil {
			if err := client.Conn.WriteJSON(message); err != nil {
				continue
			}
		}
	}

	return nil
}

func (r *memoryWebRTCRepository) SendToClient(roomID, clientID string, message interface{}) error {
	client := r.GetClient(roomID, clientID)
	if client == nil || client.Conn == nil {
		return nil
	}

	return client.Conn.WriteJSON(message)
}