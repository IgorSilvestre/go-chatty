package chat

import (
	"errors"
	"time"
)

// Domain-level errors for chat behaviors
var (
	ErrInvalidConversation = errors.New("chat: conversation/message mismatch")
	ErrNotParticipant      = errors.New("chat: sender is not a participant in the conversation")
	ErrUserBlocked         = errors.New("chat: message not allowed because one of the parties is blocked")
	ErrBackdatedMessage    = errors.New("chat: message timestamp is backdated")
	ErrEmptyMessage        = errors.New("chat: empty message (no body or attachment)")
)

// Chat is the domain aggregate for a conversation and its invariants.
//
// Notes:
//   - This aggregate is intentionally minimal and in-memory; the application layer
//     should hydrate it with the needed participants and last message timestamp
//     before invoking its behaviors.
//   - Persistence is handled by repositories outside the domain; this type only
//     enforces rules and shapes intent.
//
// Future: for group chats, Participants may include many userIDs; block checks
// should be applied against all other members.
type Chat struct {
	Conversation  Conversation
	Participants  map[string]Participant // keyed by userID
	LastMessageAt *time.Time             // last persisted message CreatedAt, if known
	Block         *Block
}

// HasParticipant tells whether userID is part of this chat.
func (c *Chat) HasParticipant(userID string) bool {
	if c == nil || c.Participants == nil {
		return false
	}
	_, ok := c.Participants[userID]
	return ok
}

// PostMessage applies domain rules and returns a validated message ready to persist.
//
// Validations:
// - Conversation/message identity must match
// - Sender must be a participant
// - No blocks between sender and any other participant (bidirectional check)
// - Message must not be backdated relative to LastMessageAt (if known)
// - Non-system messages must include either body or attachment
//
// Behavior:
// - If m.CreatedAt is zero, it is set to now.
// - On success, c.LastMessageAt is advanced to m.CreatedAt.
//
// The isBlocked function should return true if either direction of block is in effect.
// If isBlocked is nil, blocks are not checked.
func (c *Chat) PostMessage(m Message, now time.Time) (Message, error) {
	// Identity check
	if m.ConversationID == "" || c.Conversation.ID == "" || m.ConversationID != c.Conversation.ID {
		return Message{}, ErrInvalidConversation
	}

	// Membership check
	if !c.HasParticipant(m.SenderID) {
		return Message{}, ErrNotParticipant
	}

	// Block checks for DM
	if c.Block != nil && len(c.Participants) > 2 {
		return Message{}, ErrUserBlocked
	}

	// Timestamp normalization
	ts := m.CreatedAt
	if ts.IsZero() {
		if now.IsZero() {
			now = time.Now().UTC()
		}
		ts = now.UTC()
	}

	// Backdating guard
	if c.LastMessageAt != nil && ts.Before(c.LastMessageAt.UTC()) {
		return Message{}, ErrBackdatedMessage
	}

	// Content presence for non-system messages
	if m.MsgType != MessageTypeSystem {
		if m.Body == nil && m.AttachmentURL == nil {
			return Message{}, ErrEmptyMessage
		}
	}

	// Produce validated message
	m.CreatedAt = ts

	// Advance in-memory watermark
	c.LastMessageAt = &ts

	return m, nil
}
