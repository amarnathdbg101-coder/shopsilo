package auditlog

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Entry encapsulates an immutable, structured audit trail event.
// Represents: WHO did WHAT, to WHICH entity, WHEN, from WHERE, and WHAT changed.
type Entry struct {
	ID           string         `json:"id"`
	ActorID      string         `json:"actor_id,omitempty"`
	ActorEmail   string         `json:"actor_email,omitempty"`
	Action       string         `json:"action"`                  // e.g. "product.update", "user.password_reset", "order.cancelled"
	Entity       string         `json:"entity"`                  // e.g. "products", "users", "orders"
	EntityID     string         `json:"entity_id,omitempty"`     // e.g. "prod_123"
	OldState     any            `json:"old_state,omitempty"`     // State before modification (sensitive fields auto-masked)
	NewState     any            `json:"new_state,omitempty"`     // State after modification (sensitive fields auto-masked)
	IPAddress    string         `json:"ip_address,omitempty"`    // Client IP (supports IPv4 / IPv6)
	UserAgent    string         `json:"user_agent,omitempty"`    // Client browser / device user agent
	Timestamp    time.Time      `json:"timestamp"`               // UTC timestamp of event
	Metadata     map[string]any `json:"metadata,omitempty"`      // Arbitrary key-value contextual information
}

// NewEntry initializes an audit event with UTC timestamp and collision-free unique ID.
func NewEntry(action, entity, entityID string) *Entry {
	return &Entry{
		ID:        generateAuditID(),
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]any),
	}
}

func generateAuditID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("aud_%d_%s", time.Now().UnixNano(), hex.EncodeToString(b))
}
