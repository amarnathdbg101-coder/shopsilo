package gowshub

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

// Option tunes Hub behavior.
type Option func(*Hub)

// WithOnConnect registers a lifecycle callback invoked when a connection joins a room.
func WithOnConnect(fn func(room string, conn *SafeConn)) Option {
	return func(h *Hub) {
		h.onConnect = fn
	}
}

// WithOnDisconnect registers a lifecycle callback invoked when a connection leaves a room.
func WithOnDisconnect(fn func(room string, conn *SafeConn)) Option {
	return func(h *Hub) {
		h.onDisconnect = fn
	}
}

// Hub manages pub-sub WebSocket rooms, thread-safe concurrent multicasting, and ghost cleanup.
type Hub struct {
	mu           sync.RWMutex
	rooms        map[string]map[*SafeConn]struct{}
	connRooms    map[*SafeConn]map[string]struct{}
	onConnect    func(room string, conn *SafeConn)
	onDisconnect func(room string, conn *SafeConn)
	closed       atomic.Bool
}

// New creates an initialized Hub.
func New(opts ...Option) *Hub {
	h := &Hub{
		rooms:     make(map[string]map[*SafeConn]struct{}),
		connRooms: make(map[*SafeConn]map[string]struct{}),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Join adds a safe connection to the target room. O(1) membership addition.
func (h *Hub) Join(room string, conn *SafeConn) error {
	if h.closed.Load() {
		return ErrHubClosed
	}
	if room == "" {
		return ErrEmptyRoom
	}
	if conn == nil {
		return ErrNilConn
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[room]; !exists {
		h.rooms[room] = make(map[*SafeConn]struct{})
	}
	h.rooms[room][conn] = struct{}{}

	if _, exists := h.connRooms[conn]; !exists {
		h.connRooms[conn] = make(map[string]struct{})
	}
	h.connRooms[conn][room] = struct{}{}

	if h.onConnect != nil {
		go h.onConnect(room, conn)
	}

	return nil
}

// Leave removes a connection from a specific room. O(1) removal.
func (h *Hub) Leave(room string, conn *SafeConn) {
	if room == "" || conn == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, exists := h.rooms[room]; exists {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.rooms, room)
		}
	}

	if rooms, exists := h.connRooms[conn]; exists {
		delete(rooms, room)
		if len(rooms) == 0 {
			delete(h.connRooms, conn)
		}
	}

	if h.onDisconnect != nil {
		go h.onDisconnect(room, conn)
	}
}

// LeaveAll removes a connection from every room it subscribed to (e.g. on client disconnect).
func (h *Hub) LeaveAll(conn *SafeConn) {
	if conn == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	rooms, exists := h.connRooms[conn]
	if !exists {
		return
	}

	for room := range rooms {
		if conns, ok := h.rooms[room]; ok {
			delete(conns, conn)
			if len(conns) == 0 {
				delete(h.rooms, room)
			}
		}
		if h.onDisconnect != nil {
			go h.onDisconnect(room, conn)
		}
	}

	delete(h.connRooms, conn)
}

// Broadcast sends a message to all active subscribers in the specified room concurrently.
// Any connection reporting a write failure is automatically pruned to prevent ghost leaks.
func (h *Hub) Broadcast(room string, messageType int, data []byte) {
	if h.closed.Load() || room == "" || len(data) == 0 {
		return
	}

	h.mu.RLock()
	connsMap, exists := h.rooms[room]
	if !exists || len(connsMap) == 0 {
		h.mu.RUnlock()
		return
	}

	conns := make([]*SafeConn, 0, len(connsMap))
	for c := range connsMap {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		go func(sc *SafeConn) {
			if err := sc.WriteMessage(messageType, data); err != nil {
				h.Leave(room, sc)
				_ = sc.Close()
			}
		}(conn)
	}
}

// BroadcastJSON serializes an object into JSON and broadcasts it to all subscribers in the room.
func (h *Hub) BroadcastJSON(room string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	h.Broadcast(room, TextMessage, data)
	return nil
}

// BroadcastEvent serializes a standardized Event struct and broadcasts it to the room.
func (h *Hub) BroadcastEvent(room string, eventName string, payload any) error {
	evt := NewEvent(eventName, payload)
	bytes, err := evt.Bytes()
	if err != nil {
		return err
	}
	h.Broadcast(room, TextMessage, bytes)
	return nil
}

// BroadcastAll broadcasts a message to every connected client across all rooms.
func (h *Hub) BroadcastAll(messageType int, data []byte) {
	if h.closed.Load() || len(data) == 0 {
		return
	}

	h.mu.RLock()
	allConns := make([]*SafeConn, 0, len(h.connRooms))
	for conn := range h.connRooms {
		allConns = append(allConns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range allConns {
		go func(sc *SafeConn) {
			if err := sc.WriteMessage(messageType, data); err != nil {
				h.LeaveAll(sc)
				_ = sc.Close()
			}
		}(conn)
	}
}

// BroadcastAllEvent broadcasts a standardized Event across all connected clients.
func (h *Hub) BroadcastAllEvent(eventName string, payload any) error {
	evt := NewEvent(eventName, payload)
	bytes, err := evt.Bytes()
	if err != nil {
		return err
	}
	h.BroadcastAll(TextMessage, bytes)
	return nil
}

// RoomSize returns the current subscriber count for a room.
func (h *Hub) RoomSize(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
}

// Rooms returns a list of all active room names.
func (h *Hub) Rooms() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	rooms := make([]string, 0, len(h.rooms))
	for r := range h.rooms {
		rooms = append(rooms, r)
	}
	return rooms
}

// TotalConnections returns the count of unique active connections across all rooms.
func (h *Hub) TotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connRooms)
}

// Close drains and closes all active connections and cleans up resources.
func (h *Hub) Close() error {
	if !h.closed.CompareAndSwap(false, true) {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.connRooms {
		_ = conn.Close()
	}

	h.rooms = make(map[string]map[*SafeConn]struct{})
	h.connRooms = make(map[*SafeConn]map[string]struct{})
	return nil
}
