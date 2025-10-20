package usecase

import (
	"context"
	"fmt"
	chat "go-chatty/internal/pkg/chat/application/domain"
	repository "go-chatty/internal/pkg/chat/persistence/repository/port"
	"time"
)

// CreateChatInput carries the required data to open a new conversation
// Note: variables with multiple items use plural naming per guideline
// Domain constraints for participants and block logic are minimal for now
// (no dedup of existing conversations, etc.).
type CreateChatInput struct {
	TenantID       string
	ParticipantIDs []string
}

// CreateChatUseCase handles creation of a new conversation and its participants
// Hexagonal: depends on repository port only
// One class per use case (own file)
type CreateChatUseCase struct {
	Repo repository.ChatRepository
}

func NewCreateChatUseCase(repo repository.ChatRepository) *CreateChatUseCase {
	return &CreateChatUseCase{Repo: repo}
}

// Execute persists a conversation and registers participants
func (uc *CreateChatUseCase) Execute(ctx context.Context, in CreateChatInput) (*chat.Conversation, error) {
	if in.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if len(in.ParticipantIDs) == 0 {
		return nil, fmt.Errorf("participant_ids must include at least one user id")
	}

	now := time.Now().UTC()
	conv := chat.Conversation{CreatedAt: now, TenantID: in.TenantID}

	id, err := uc.Repo.CreateConversation(ctx, conv)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersistence, err)
	}
	conv.ID = id

	for _, uid := range in.ParticipantIDs {
		if uid == "" {
			continue
		}
		p := chat.Participant{
			ConversationID: id,
			UserID:         uid,
			Role:           chat.ParticipantRoleMember,
		}
		if err := uc.Repo.AddParticipant(ctx, p); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPersistence, err)
		}
	}

	return &conv, nil
}
