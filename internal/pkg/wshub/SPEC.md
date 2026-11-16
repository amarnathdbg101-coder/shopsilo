# Technical Specification: go-wshub

## 1. Domain & Purpose
`go-wshub` is a high-throughput, thread-safe WebSocket connection hub, pub-sub room manager, and keepalive heartbeat engine designed for real-time Go architectures. It eliminates WebSocket concurrent write race panics, provides O(1) room membership, and automatically evicts ghost connections upon network dropouts.

## 2. Architecture & Design Principles
- **Concurrent Write Serialization**: Gorilla WebSocket connections trigger internal panics (`concurrent write to websocket connection`) if two goroutines invoke `WriteMessage` simultaneously. `SafeConn` wraps the transport with a dedicated write mutex and automatic deadline timeouts.
- **O(1) Pub-Sub Room Routing**: Stores connection mappings using dual-index hash tables (`rooms[roomName][conn]` and `connRooms[conn][roomName]`), ensuring room joins, leaves, and total disconnections execute in amortized O(1) time without slice iteration.
- **Auto Ghost Connection Pruning**: When mobile clients lose network coverage or background apps without terminating TCP handshakes, write errors automatically invoke `Leave(room, conn)` and close the connection, eliminating memory leaks.
- **Zero Third-Party Dependencies**: 100% pure Go standard library (`sync`, `time`, `encoding/json`, `context`). Can wrap Gorilla WebSocket, Nhooyr WebSocket, or custom TCP mock connections seamlessly.

## 3. API Surface
- `type UnderlyingConn interface`
- `type SafeConn struct`
- `func NewSafeConn(conn UnderlyingConn, opts ...ConnOption) *SafeConn`
- `func (s *SafeConn) WriteMessage(messageType int, data []byte) error`
- `func (s *SafeConn) WriteJSON(v any) error`
- `func (s *SafeConn) WriteControl(messageType int, data []byte, deadline time.Time) error`
- `func (s *SafeConn) Close() error`
- `type Event struct`
- `func NewEvent(name string, payload any) *Event`
- `type Hub struct`
- `func New(opts ...Option) *Hub`
- `func (h *Hub) Join(room string, conn *SafeConn) error`
- `func (h *Hub) Leave(room string, conn *SafeConn)`
- `func (h *Hub) LeaveAll(conn *SafeConn)`
- `func (h *Hub) Broadcast(room string, messageType int, data []byte)`
- `func (h *Hub) BroadcastJSON(room string, v any) error`
- `func (h *Hub) BroadcastEvent(room string, eventName string, payload any) error`
- `func (h *Hub) BroadcastAll(messageType int, data []byte)`
- `func (h *Hub) BroadcastAllEvent(eventName string, payload any) error`
- `func (h *Hub) RoomSize(room string) int`
- `func (h *Hub) Rooms() []string`
- `func (h *Hub) TotalConnections() int`
- `func (h *Hub) Close() error`
- `func StartHeartbeat(ctx context.Context, conn *SafeConn, pingInterval, writeTimeout time.Duration) func()`
