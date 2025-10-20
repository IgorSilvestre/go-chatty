package chat

import "time"

// Block represents a 1:1 block (future-proof for groups)
type Block struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	BlockerId string    `db:"blocker_id"`
	BlockedId string    `db:"blocked_id"`
}
