package usecase

import (
	"context"
	"fmt"

	repository "go-chatty/internal/pkg/chat/persistence/repository/port"
)

// ListParticipantsInput wraps the conversation identifier to fetch its participants.
type ListParticipantsInput struct {
	ConversationID string
}

// ListParticipantsUseCase returns user IDs for all participants in the conversation.
type ListParticipantsUseCase struct {
	Repo repository.ChatRepository
}

func NewListParticipantsUseCase(repo repository.ChatRepository) *ListParticipantsUseCase {
	return &ListParticipantsUseCase{Repo: repo}
}

func (uc *ListParticipantsUseCase) Execute(ctx context.Context, in ListParticipantsInput) ([]string, error) {
	if in.ConversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
	}

	ids, err := uc.Repo.ListParticipantIDs(ctx, in.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersistence, err)
	}
	return ids, nil
}
