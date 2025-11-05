package repositories

import (
	"live-shopping-ai/backend/internal/domain/entities"
)

type WebRTCRepository interface {
	CreateRoom(roomID string) *entities.Room
	GetRoom(roomID string) *entities.Room
	RemoveRoom(roomID string)
	AddClientToRoom(roomID string, client *entities.Client)
	RemoveClientFromRoom(roomID string, clientID string)
	GetClient(roomID, clientID string) *entities.Client
	GetRoomClients(roomID string) []*entities.Client
	BroadcastToRoom(roomID string, message interface{}, excludeClientID string) error
	SendToClient(roomID, clientID string, message interface{}) error
}