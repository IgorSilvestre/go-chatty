package chat

import "time"

// ParticipantRole expresses the role within a conversation
// 0 = member (default); extra values reserved for future group roles
type ParticipantRole int16

const (
	ParticipantRoleMember ParticipantRole = 0
)

// Participant captures membership and read/mute state
// Primary key: (ConversationID, UserID)
type Participant struct {
	ConversationID string          `db:"conversation_id"`
	UserID         string          `db:"user_id"`
	Role           ParticipantRole `db:"role"`
	LastReadMsg    *string         `db:"last_read_msg"`
	MutedUntil     *time.Time      `db:"muted_until"`
}
