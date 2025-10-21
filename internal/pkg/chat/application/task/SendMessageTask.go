package task

import (
	"context"
	"encoding/json"
	"time"

	qport "go-chatty/internal/infrastructure/queue/port"
	chat "go-chatty/internal/pkg/chat/application/domain"
	"go-chatty/internal/pkg/chat/application/usecase"
	repoAdapter "go-chatty/internal/pkg/chat/persistence/repository/adapter"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SendMessageTaskType is the queue task name for sending a message within the chat domain.
const SendMessageTaskType = "chat:send_message"

// SendMessageTaskPayload is the JSON payload transported via the queue.
// Kept decoupled from domain types to avoid tight coupling with JSON tags.
type SendMessageTaskPayload struct {
	ConversationID string  `json:"conversationId"`
	SenderID       string  `json:"senderId"`
	Body           *string `json:"body"`
	MsgType        int16   `json:"msgType"`
	AttachmentURL  *string `json:"attachmentUrl"`
	AttachmentMeta *string `json:"attachmentMeta"`
	DedupeKey      *string `json:"dedupeKey"`
}

// RegisterSendMessageTask binds the task handler to the provided server.
// The handler will execute the SendMessageUseCase using the provided DB pool.
func RegisterSendMessageTask(srv qport.Server, pool *pgxpool.Pool) {
	srv.Register(SendMessageTaskType, func(ctx context.Context, t qport.Task) error {
		var p SendMessageTaskPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			// malformed payload: do not retry indefinitely
			return err
		}

		// Construct use case with repository adapter
		repo := repoAdapter.NewPgChatRepository(pool)
		uc := usecase.NewSendMessageUseCase(repo)

		in := usecase.SendMessageInput{
			ConversationID: p.ConversationID,
			SenderID:       p.SenderID,
			Body:           p.Body,
			MsgType:        chat.MessageType(p.MsgType),
			AttachmentURL:  p.AttachmentURL,
			AttachmentMeta: p.AttachmentMeta,
			DedupeKey:      p.DedupeKey,
		}

		// give DB a reasonable time budget per task execution
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		_, err := uc.Execute(ctx, in)
		if err != nil {
			// If the error is a persistence error, signal retry; otherwise also return error
			// The retry/backoff policy is controlled by the adapter/server.
			return err
		}
		return nil
	})
}
