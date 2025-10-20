package repository

import (
	"context"
	"go-chatty/internal/pkg/chat/application/domain"
	"time"
)

// ChatRepository defines persistence operations for the chat domain
// Note: Receipt and Block operations were removed from Chat; handle them in a separate context/service if needed.
type ChatRepository interface {
	CreateConversation(ctx context.Context, c chat.chat) error
	AddParticipant(ctx context.Context, p chat.Participant) error
	SaveMessage(ctx context.Context, m chat.Message) error
	GetMessagesByConversation(ctx context.Context, conversationID string, limit int, offset int) ([]chat.Message, error)
	UpdateParticipantReadState(ctx context.Context, conversationID string, userID string, lastReadMsg *string) error
	SetMuteUntil(ctx context.Context, conversationID string, userID string, mutedUntil *time.Time) error
}
