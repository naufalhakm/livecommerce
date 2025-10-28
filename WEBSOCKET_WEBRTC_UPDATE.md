# WebSocket & WebRTC Implementation Update

## Overview
Updated the WebSocket and WebRTC implementation to support room-based video streaming where sellers broadcast to viewers in specific rooms.

## Changes Made

### Backend (Golang)

#### 1. WebSocket Service (`backend/internal/services/websocket.go`)
- **Room-based Architecture**: Changed from global broadcast to room-based messaging
- **Required Parameters**: Both `client_id` and `room_id` are now required to connect
- **Hub Structure**: 
  - Replaced `clients map[*Client]bool` with `rooms map[string]map[*Client]bool`
  - Each room contains multiple clients
- **Client Structure**: Added `roomID string` field
- **New Methods**:
  - `BroadcastToRoom(roomID, message)`: Broadcast to all clients in a specific room
  - `SendToClient(roomID, clientID, message)`: Send message to specific client in a room
- **Message Handling**:
  - WebRTC signaling (offer, answer, ICE candidates) can be sent to specific clients or broadcast to room
  - Chat messages are broadcast to the entire room
  - All messages include `from` and `room_id` fields
- **User Events**: Automatically notify room when users join/leave

#### 2. Stream Handler (`backend/internal/api/stream_handler.go`)
- Updated to use `BroadcastToRoom()` instead of `BroadcastMessage()`
- Stream status changes broadcast to seller's room (using seller_id as room_id)
- Product detection results broadcast to seller's room

### Frontend (React)

#### 1. WebSocket Service (`frontend/src/services/websocket.js`)
- **Updated `connect()` method**: Now requires both `clientId` and `roomId` parameters
- **Connection URL**: `ws://localhost:8080/ws/livestream?client_id={clientId}&room_id={roomId}`
- **New Method**: `sendChat(message, username)` for sending chat messages

#### 2. WebRTC Service (`frontend/src/services/webrtc.js`)
- **ICE Trickle**: Enabled trickle ICE for better connection establishment
- **STUN Servers**: Added Google STUN servers for NAT traversal
- **Updated `createPeer()` method**: Now accepts `targetClientId` parameter for directed signaling
- **New Method**: `setupSignalingListeners()` to handle WebRTC signaling messages
- **Updated Methods**:
  - `startBroadcast()`: For sellers to start broadcasting
  - `joinBroadcast(sellerClientId)`: For viewers to join and receive stream

#### 3. Seller Page (`frontend/src/pages/LiveStreamSeller.jsx`)
- **WebSocket Connection**: `websocketService.connect(`seller-${sellerId}`, sellerId)`
  - `client_id`: `seller-{sellerId}` (e.g., "seller-1")
  - `room_id`: `{sellerId}` (e.g., "1")
- **WebRTC**: Calls `webrtcService.startBroadcast()` to start broadcasting video
- **Viewer Tracking**: Listens to `user_joined` and `user_left` events to update viewer count
- **Product Pinning**: Broadcasts pinned products to all viewers in the room

#### 4. Viewer Page (`frontend/src/pages/LiveStreamViewer.jsx`)
- **WebSocket Connection**: `websocketService.connect(`viewer-${timestamp}`, sellerId)`
  - `client_id`: `viewer-{timestamp}` (e.g., "viewer-1761553747061")
  - `room_id`: `{sellerId}` (e.g., "1")
- **WebRTC**: Calls `webrtcService.joinBroadcast(`seller-${sellerId}`)` to receive video stream
- **Video Display**: Shows remote stream from seller in video element
- **Chat**: Uses `websocketService.sendChat()` to send messages to the room
- **Product Updates**: Receives pinned products and detection results from seller

## Connection Flow

### Seller Flow
1. Seller enters their seller ID (e.g., "1")
2. Clicks "Start Livestream"
3. Camera initializes
4. WebSocket connects: `client_id=seller-1&room_id=1`
5. WebRTC starts broadcasting
6. Seller is now live in room "1"

### Viewer Flow
1. Viewer enters seller ID (e.g., "1")
2. Clicks "Join Stream"
3. WebSocket connects: `client_id=viewer-{timestamp}&room_id=1`
4. WebRTC joins broadcast from `seller-1`
5. Viewer receives video stream
6. Viewer can chat and see pinned products

## Message Types

### WebRTC Signaling
- `webrtc_offer`: Sent by seller to initiate connection
- `webrtc_answer`: Sent by viewer in response to offer
- `webrtc_ice_candidate`: ICE candidates for NAT traversal

### Room Events
- `user_joined`: Broadcast when a user joins the room
- `user_left`: Broadcast when a user leaves the room
- `seller_live`: Seller announces they are live
- `seller_offline`: Seller announces they are offline

### Content
- `chat`: Chat messages with username and message
- `pin_product`: Seller pins a product to display
- `product_detection`: AI-detected products from video frame
- `reaction`: Viewer reactions (❤️, 👍, 🔥, 🎉)

## Room ID Convention
- Room ID = Seller ID
- Example: Seller with ID "1" creates room "1"
- All viewers join room "1" to watch seller "1"

## Client ID Convention
- Seller: `seller-{sellerId}` (e.g., "seller-1")
- Viewer: `viewer-{timestamp}` (e.g., "viewer-1761553747061")

## Testing

### Start Backend
```bash
cd backend
go run cmd/main.go
```

### Test Seller
1. Open browser: http://localhost:3000/seller
2. Enter seller ID: 1
3. Click "Create Stream Room"
4. Click "Start Livestream"
5. Allow camera access

### Test Viewer
1. Open another browser/tab: http://localhost:3000/viewer
2. Enter seller ID: 1
3. Click "Join Stream"
4. Should see seller's video stream

### Test Chat
1. As viewer, type message and press Enter
2. Message should appear in chat (currently only on viewer side, needs backend relay)

## Next Steps
1. Add chat message relay through backend
2. Implement reaction animations
3. Add viewer list display
4. Handle reconnection logic
5. Add stream quality selection
6. Implement recording functionality
