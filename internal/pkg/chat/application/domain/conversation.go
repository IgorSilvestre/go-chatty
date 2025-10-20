package chat

import "time"

// Conversation represents a 1:1 thread (future-proof for groups)
type Conversation struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	TenantID  string    `db:"tenant_id"`
}
