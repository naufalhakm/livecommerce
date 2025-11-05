package services

import (
	"fmt"
	"live-shopping-ai/backend/internal/domain/entities"
	"live-shopping-ai/backend/internal/domain/repositories"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type WebRTCService interface {
	HandleWebSocketConnection(conn *websocket.Conn) error
	HandleClientJoin(roomID, clientID, role string, conn *websocket.Conn) error
	HandleOffer(roomID, clientID string, offer webrtc.SessionDescription, targetClientID string) error
	HandleAnswer(roomID, clientID string, answer webrtc.SessionDescription, targetClientID string) error
	HandleICECandidate(roomID, clientID string, candidateData map[string]interface{}, targetClientID string) error
	CreatePeerConnection(role string) (*webrtc.PeerConnection, error)
	CleanupRoom(roomID string)
	GetRoomStats(roomID string) map[string]interface{}
}

type webrtcService struct {
	repo            repositories.WebRTCRepository
	liveStreamRepo  repositories.LiveStreamRepository
	config          entities.WebRTCConfig
	roomsMutex      sync.RWMutex
	cleanupTime     time.Duration
}

func NewWebRTCService(repo repositories.WebRTCRepository, liveStreamRepo repositories.LiveStreamRepository) WebRTCService {
	stunServers := []string{
		"stun:stun.l.google.com:19302",
		"stun:stun1.l.google.com:19302",
		"stun:stun2.l.google.com:19302",
		"stun:stun3.l.google.com:19302",
		"stun:stun4.l.google.com:19302",
		"stun:global.stun.twilio.com:3478",
	}

	iceServers := []webrtc.ICEServer{}
	for _, server := range stunServers {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs: []string{server},
		})
	}

	turnURL := os.Getenv("TURN_SERVER_URL")
	if turnURL != "" {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       []string{turnURL},
			Username:   os.Getenv("TURN_USERNAME"),
			Credential: os.Getenv("TURN_PASSWORD"),
		})
	}

	config := entities.WebRTCConfig{
		ICEServers:   iceServers,
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	}

	return &webrtcService{
		repo:           repo,
		liveStreamRepo: liveStreamRepo,
		config:         config,
		cleanupTime:    1 * time.Hour,
	}
}

func (s *webrtcService) HandleWebSocketConnection(conn *websocket.Conn) error {
	var roomID, clientID string

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		// Track client info for cleanup
		if msgType, ok := msg["type"].(string); ok && msgType == "join" {
			roomID, _ = msg["room"].(string)
			if data, ok := msg["data"].(map[string]interface{}); ok {
				clientID, _ = data["client_id"].(string)
			}
		}

		if err := s.handleMessage(conn, msg); err != nil {
		}
	}

	// Cleanup when connection closes
	if roomID != "" && clientID != "" {
		s.cleanupClient(roomID, clientID)
	}

	return nil
}

func (s *webrtcService) handleMessage(conn *websocket.Conn, msg map[string]interface{}) error {
	msgType, ok := msg["type"].(string)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	
	roomID, _ := msg["room"].(string)
	clientID, _ := msg["client_id"].(string)
	fromClientID, _ := msg["from"].(string)
	toClientID, _ := msg["to"].(string)


	switch msgType {
	case "join":
		data, ok := msg["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid join data")
		}
		clientID, ok := data["client_id"].(string)
		if !ok {
			return fmt.Errorf("missing client_id")
		}
		role, ok := data["role"].(string)
		if !ok {
			return fmt.Errorf("missing role")
		}
		return s.HandleClientJoin(roomID, clientID, role, conn)

	case "webrtc_offer":
		data, ok := msg["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid offer data")
		}
		sdp, ok := data["sdp"].(string)
		if !ok {
			return fmt.Errorf("missing SDP")
		}
		offer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  sdp,
		}
		return s.HandleOffer(roomID, fromClientID, offer, toClientID)

	case "webrtc_answer":
		data, ok := msg["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid answer data")
		}
		sdp, ok := data["sdp"].(string)
		if !ok {
			return fmt.Errorf("missing SDP")
		}
		answer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  sdp,
		}
		return s.HandleAnswer(roomID, fromClientID, answer, toClientID)

	case "webrtc_ice_candidate":
		data, ok := msg["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid ICE data")
		}
		
		
		// Simply forward the ICE candidate data as-is to maintain browser compatibility
		return s.HandleICECandidate(roomID, fromClientID, data, toClientID)

	case "chat":
		message := entities.WebRTCMessage{
			Type: "chat",
			Data: msg["data"],
			Room: roomID,
			From: clientID,
		}
		return s.repo.BroadcastToRoom(roomID, message, "")

	case "reaction":
		message := entities.WebRTCMessage{
			Type: "reaction",
			Data: msg["data"],
			Room: roomID,
			From: clientID,
		}
		return s.repo.BroadcastToRoom(roomID, message, "")

	case "seller_live":
		// Just broadcast that seller is live - no peer connection handling needed
		message := entities.WebRTCMessage{
			Type: "seller_live",
			Data: msg["data"],
			Room: roomID,
		}
		return s.repo.BroadcastToRoom(roomID, message, clientID)

	case "seller_offline":
		message := entities.WebRTCMessage{
			Type: "seller_offline",
			Data: msg["data"],
			Room: roomID,
		}
		return s.repo.BroadcastToRoom(roomID, message, clientID)

	default:
	}

	return nil
}

func (s *webrtcService) HandleClientJoin(roomID, clientID, role string, conn *websocket.Conn) error {

	// For pure signaling server, we don't create peer connections on backend
	client := &entities.Client{
		ID:            clientID,
		Conn:          conn,
		PeerConnection: nil, // No backend peer connection needed
		Role:          role,
		RoomID:        roomID,
		ConnectedAt:   time.Now(),
	}

	s.repo.AddClientToRoom(roomID, client)

	// Send joined message immediately
	response := entities.WebRTCMessage{
		Type: "joined",
		Data: map[string]string{"status": "success", "client_id": clientID},
		Room: roomID,
	}
	if err := conn.WriteJSON(response); err != nil {
		return err
	}

	userJoinMsg := entities.WebRTCMessage{
		Type: "user_joined",
		Data: map[string]string{"client_id": clientID},
		Room: roomID,
	}
	s.repo.BroadcastToRoom(roomID, userJoinMsg, clientID)

	// Update viewer count for livestream
	if role == "viewer" {
		s.updateViewerCount(roomID)
	}

	return nil
}

// No longer needed - backend is pure signaling server



func (s *webrtcService) HandleOffer(roomID, fromClientID string, offer webrtc.SessionDescription, toClientID string) error {
	// Forward the offer to the seller
	message := entities.WebRTCMessage{
		Type: "webrtc_offer",
		Data: map[string]string{"sdp": offer.SDP},
		Room: roomID,
		From: fromClientID,
		To:   toClientID,
	}

	return s.repo.SendToClient(roomID, toClientID, message)
}

func (s *webrtcService) HandleAnswer(roomID, fromClientID string, answer webrtc.SessionDescription, toClientID string) error {
	// Forward the answer to the target client
	message := entities.WebRTCMessage{
		Type: "webrtc_answer",
		Data: map[string]string{"sdp": answer.SDP},
		Room: roomID,
		From: fromClientID,
		To:   toClientID,
	}

	return s.repo.SendToClient(roomID, toClientID, message)
}

func (s *webrtcService) HandleICECandidate(roomID, fromClientID string, candidateData map[string]interface{}, toClientID string) error {

	// Forward the ICE candidate data as-is to maintain browser compatibility
	message := entities.WebRTCMessage{
		Type: "webrtc_ice_candidate",
		Data: candidateData,
		Room: roomID,
		From: fromClientID,
		To:   toClientID,
	}

	return s.repo.SendToClient(roomID, toClientID, message)
}

func (s *webrtcService) CreatePeerConnection(role string) (*webrtc.PeerConnection, error) {
	config := webrtc.Configuration{
		ICEServers:   s.config.ICEServers,
		SDPSemantics: s.config.SDPSemantics,
	}

	mediaEngine := &webrtc.MediaEngine{}
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}

	// Add support for H.264 for better compatibility
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
		MimeType:    webrtc.MimeTypeH264,
		ClockRate:   90000,
		Channels:    0,
		SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
		},
		PayloadType: 102,
	}, webrtc.RTPCodecTypeVideo); err != nil {
	}

	settingEngine := webrtc.SettingEngine{}
	
	// Set UDP port range for better NAT traversal
	settingEngine.SetEphemeralUDPPortRange(50000, 60000)
	
	// Enable ICE TCP (helps with some network configurations)
	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
		webrtc.NetworkTypeTCP4,
		webrtc.NetworkTypeTCP6,
	})

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
	)

	pc, err := api.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	return pc, nil
}



func (s *webrtcService) cleanupClient(roomID, clientID string) {
	client := s.repo.GetClient(roomID, clientID)
	if client == nil {
		return
	}
	
	userLeftMsg := entities.WebRTCMessage{
		Type: "user_left",
		Data: map[string]string{"client_id": clientID},
		Room: roomID,
	}
	s.repo.BroadcastToRoom(roomID, userLeftMsg, clientID)

	s.repo.RemoveClientFromRoom(roomID, clientID)
	
	// Update viewer count if it was a viewer
	if client.Role == "viewer" {
		s.updateViewerCount(roomID)
	}
	
	room := s.repo.GetRoom(roomID)
	if room != nil && len(room.Clients) == 0 {
		time.AfterFunc(s.cleanupTime, func() {
			if s.repo.GetRoom(roomID) != nil && len(s.repo.GetRoom(roomID).Clients) == 0 {
				s.repo.RemoveRoom(roomID)
			}
		})
	}
}

func (s *webrtcService) CleanupRoom(roomID string) {
	s.repo.RemoveRoom(roomID)
}

func (s *webrtcService) GetRoomStats(roomID string) map[string]interface{} {
	room := s.repo.GetRoom(roomID)
	if room == nil {
		return nil
	}

	room.Mutex.RLock()
	defer room.Mutex.RUnlock()

	stats := map[string]interface{}{
		"room_id":    roomID,
		"clients":    len(room.Clients),
		"publishers": 0,
		"viewers":    0,
	}

	for _, client := range room.Clients {
		if client.Role == "publisher" {
			stats["publishers"] = stats["publishers"].(int) + 1
		} else {
			stats["viewers"] = stats["viewers"].(int) + 1
		}
	}

	return stats
}

func (s *webrtcService) updateViewerCount(roomID string) {
	room := s.repo.GetRoom(roomID)
	if room == nil {
		return
	}
	
	room.Mutex.RLock()
	viewerCount := 0
	for _, client := range room.Clients {
		if client.Role == "viewer" {
			viewerCount++
		}
	}
	room.Mutex.RUnlock()
	
	// Update viewer count in database
	if err := s.liveStreamRepo.UpdateViewerCount(roomID, viewerCount); err != nil {
	}
	
}