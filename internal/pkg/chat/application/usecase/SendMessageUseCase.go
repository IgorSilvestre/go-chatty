package usecase

import (
	"context"
	"fmt"
	chat "go-chatty/internal/pkg/chat/application/domain"
	repository "go-chatty/internal/pkg/chat/persistence/repository/port"
)

// SendMessageInput carries the data needed to send a new message
// Note: Validation for body/attachment and defaults are handled in controller/usecase layers
// to preserve domain integrity via chat.NewMessage.
type SendMessageInput struct {
	ConversationID string
	SenderID       string
	Body           *string
	MsgType        chat.MessageType
	AttachmentURL  *string
	AttachmentMeta *string
	DedupeKey      *string
}

// SendMessageUseCase handles the SendMessage application service
// Hexagonal: depends on repository port, returns domain entity
// One class per use case (own file)
type SendMessageUseCase struct {
	Repo repository.ChatRepository
}

func NewSendMessageUseCase(repo repository.ChatRepository) *SendMessageUseCase {
	return &SendMessageUseCase{Repo: repo}
}

// Execute sends/persists a new message for a conversation
func (uc *SendMessageUseCase) Execute(ctx context.Context, in SendMessageInput) (*chat.Message, error) {
	if in.ConversationID == "" || in.SenderID == "" {
		return nil, fmt.Errorf("conversationId and senderId are required")
	}

	isParticipant, err := uc.Repo.IsParticipant(ctx, in.ConversationID, in.SenderID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersistence, err)
	}
	if !isParticipant {
		return nil, chat.ErrNotParticipant
	}

	msgInput := chat.Message{
		ConversationID: in.ConversationID,
		SenderID:       in.SenderID,
		Body:           in.Body,
		MsgType:        in.MsgType,
		AttachmentURL:  in.AttachmentURL,
		AttachmentMeta: in.AttachmentMeta,
		DedupeKey:      in.DedupeKey,
	}

	msg, err := chat.NewMessage(msgInput)
	if err != nil {
		return nil, err
	}

	// Persist letting DB generate the ID
	id, err := uc.Repo.SaveMessage(ctx, *msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersistence, err)
	}
	msg.ID = id
	return msg, nil
}
