package chat

import (
	"errors"
	"strings"
	"time"
)

// MessageType represents type of message content
// 0=text, 1=image, 2=file, 3=system
type MessageType int16

const (
	MessageTypeText   MessageType = 0
	MessageTypeImage  MessageType = 1
	MessageTypeFile   MessageType = 2
	MessageTypeSystem MessageType = 3
)

// Message is an immutable log entry in a conversation
type Message struct {
	ID             string      `db:"id"`
	ConversationID string      `db:"conversation_id"`
	SenderID       string      `db:"sender_id"`
	CreatedAt      time.Time   `db:"created_at"`
	Body           *string     `db:"body"`
	MsgType        MessageType `db:"msg_type"`
	AttachmentURL  *string     `db:"attachment_url"`
	AttachmentMeta *string     `db:"attachment_meta"` // JSON string; nil if absent
	DedupeKey      *string     `db:"dedupe_key"`
}

func NewMessage(m Message) (*Message, error) {
	if m.ConversationID == "" || m.SenderID == "" {
		return nil, errors.New("conversation_id and sender_id are required")
	}

	if m.Body != nil {
		trimmed := strings.TrimSpace(*m.Body)
		if trimmed == "" {
			m.Body = nil
		} else {
			m.Body = &trimmed
		}
	}

	if m.Body == nil && m.AttachmentURL == nil {
		return nil, errors.New("message must contain either body or attachment")
	}

	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}

	return &m, nil
}
