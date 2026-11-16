package auditlog

import "context"

// Store defines the persistence contract for audit entries.
// PostgreSQL, In-Memory RAM, Elasticsearch, ya Cloud SQL ke liye ye interface implement hota hai.
type Store interface {
	// SaveBatch bulk-inserts a slice of audit log entries efficiently in a single operation.
	SaveBatch(ctx context.Context, entries []*Entry) error

	// Query retrieves historical audit trail records matching filter criteria.
	Query(ctx context.Context, filter Filter) ([]*Entry, error)

	// Close terminates any active connections or background pool resources.
	Close() error
}

// Filter defines search constraints when querying audit trail history.
type Filter struct {
	Entity   string `json:"entity,omitempty"`
	EntityID string `json:"entity_id,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
	Action   string `json:"action,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}
