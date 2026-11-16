package gowshub

import (
	"encoding/json"
	"sync"
	"time"
)

// RFC 6455 WebSocket frame opcode constants matching Gorilla WebSocket defaults.
const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

// UnderlyingConn defines the minimal contract needed for WebSocket transmissions.
// Both *websocket.Conn (gorilla) and custom mock implementations satisfy this.
type UnderlyingConn interface {
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
	Close() error
}

// Deadliner is optionally satisfied by transports supporting write timeouts.
type Deadliner interface {
	SetWriteDeadline(t time.Time) error
}

// SafeConn wraps an UnderlyingConn with a dedicated write mutex, avoiding "concurrent write to websocket connection" panics.
type SafeConn struct {
	conn          UnderlyingConn
	writeMu       sync.Mutex
	writeDeadline time.Duration
	id            string
	metaMu        sync.RWMutex
	metadata      map[string]any
}

// NewSafeConn wraps a raw connection with thread-safe write protection and default 5-second deadline.
func NewSafeConn(conn UnderlyingConn, opts ...ConnOption) *SafeConn {
	sc := &SafeConn{
		conn:          conn,
		writeDeadline: 5 * time.Second,
		metadata:      make(map[string]any),
	}
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}

// ConnOption configures SafeConn parameters.
type ConnOption func(*SafeConn)

// WithWriteDeadline customizes the write deadline for outbound frames.
func WithWriteDeadline(d time.Duration) ConnOption {
	return func(s *SafeConn) {
		if d > 0 {
			s.writeDeadline = d
		}
	}
}

// WithID sets a distinct identifier for this client.
func WithID(id string) ConnOption {
	return func(s *SafeConn) {
		s.id = id
	}
}

// ID returns the connection's unique identifier.
func (s *SafeConn) ID() string {
	return s.id
}

// SetID updates the connection's identifier.
func (s *SafeConn) SetID(id string) {
	s.id = id
}

// SetMetadata stores arbitrary key-value context on this connection.
func (s *SafeConn) SetMetadata(key string, val any) {
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	s.metadata[key] = val
}

// GetMetadata retrieves context stored on this connection.
func (s *SafeConn) GetMetadata(key string) (any, bool) {
	s.metaMu.RLock()
	defer s.metaMu.RUnlock()
	val, ok := s.metadata[key]
	return val, ok
}

// Raw returns the underlying connection transport.
func (s *SafeConn) Raw() UnderlyingConn {
	return s.conn
}

// WriteMessage sends a data frame (Text/Binary) safely under write mutex lock with timeout.
func (s *SafeConn) WriteMessage(messageType int, data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.conn == nil {
		return ErrNilConn
	}

	if s.writeDeadline > 0 {
		if dl, ok := s.conn.(Deadliner); ok {
			_ = dl.SetWriteDeadline(time.Now().Add(s.writeDeadline))
		}
	}

	return s.conn.WriteMessage(messageType, data)
}

// WriteJSON marshals an object into JSON and writes it as TextMessage frame.
func (s *SafeConn) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.WriteMessage(TextMessage, data)
}

// WriteControl sends a control frame (Ping/Pong/Close) safely under write mutex lock.
func (s *SafeConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.conn == nil {
		return ErrNilConn
	}

	return s.conn.WriteControl(messageType, data, deadline)
}

// Close terminates the underlying connection under write lock.
func (s *SafeConn) Close() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if s.conn == nil {
		return nil
	}
	return s.conn.Close()
}
