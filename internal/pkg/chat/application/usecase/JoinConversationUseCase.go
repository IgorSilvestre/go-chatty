package usecase

import (
	"context"
	"fmt"

	chat "go-chatty/internal/pkg/chat/application/domain"
	repository "go-chatty/internal/pkg/chat/persistence/repository/port"
)

// JoinConversationInput validates a request to attach a user session to a conversation.
type JoinConversationInput struct {
	ConversationID string
	UserID         string
}

// JoinConversationUseCase ensures the user belongs to the conversation before joining the realtime room.
type JoinConversationUseCase struct {
	Repo repository.ChatRepository
}

func NewJoinConversationUseCase(repo repository.ChatRepository) *JoinConversationUseCase {
	return &JoinConversationUseCase{Repo: repo}
}

func (uc *JoinConversationUseCase) Execute(ctx context.Context, in JoinConversationInput) error {
	if in.ConversationID == "" || in.UserID == "" {
		return fmt.Errorf("conversation_id and user_id are required")
	}

	ok, err := uc.Repo.IsParticipant(ctx, in.ConversationID, in.UserID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPersistence, err)
	}
	if !ok {
		return chat.ErrNotParticipant
	}
	return nil
}
