package usecase

import (
	"context"
	"fmt"
	chat "go-chatty/internal/pkg/chat/application/domain"
	repository "go-chatty/internal/pkg/chat/persistence/repository/port"
)

// GetMessageInput carries parameters to fetch messages of a conversation
// Using singular naming for the use case per guideline
// Variables with multiple items use plural (e.g., messages slice in the return)
type GetMessageInput struct {
	ConversationID string
	Limit          int
	Offset         int
}

// GetMessageUseCase fetches messages for a given conversation
// Hexagonal: depends only on repository port
// One class per use case (own file)
type GetMessageUseCase struct {
	Repo repository.ChatRepository
}

func NewGetMessageUseCase(repo repository.ChatRepository) *GetMessageUseCase {
	return &GetMessageUseCase{Repo: repo}
}

// Execute returns messages for the conversation honoring limit/offset
func (uc *GetMessageUseCase) Execute(ctx context.Context, in GetMessageInput) ([]chat.Message, error) {
	if in.ConversationID == "" {
		return nil, fmt.Errorf("conversationId is required")
	}
	msgs, err := uc.Repo.GetMessagesByConversation(ctx, in.ConversationID, in.Limit, in.Offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersistence, err)
	}
	return msgs, nil
}
