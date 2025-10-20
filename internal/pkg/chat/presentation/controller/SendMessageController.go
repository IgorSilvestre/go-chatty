package controller

import (
	"context"
	"encoding/json"
	"go-chatty/internal/pkg/chat/application/task"
	"net/http"
	"time"

	queueport "go-chatty/internal/infrastructure/queue/port"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SendMessageController handles the send-message endpoint only (one controller per endpoint)
type SendMessageController struct {
	Q    queueport.Client
	pool *pgxpool.Pool // kept for future use / parity; not used here
}

func NewSendMessageController(pool *pgxpool.Pool, client queueport.Client) *SendMessageController {
	return &SendMessageController{Q: client, pool: pool}
}

// sendMessageRequest is the DTO for the HTTP request body
type sendMessageRequest struct {
	SenderID       string  `json:"sender_id" binding:"required"`
	Body           *string `json:"body"`
	MsgType        *int16  `json:"msg_type"`
	AttachmentURL  *string `json:"attachment_url"`
	AttachmentMeta *string `json:"attachment_meta"`
	DedupeKey      *string `json:"dedupe_key"`
}

// Handle returns a gin handler that enqueues a background task to send a message
func (h *SendMessageController) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		chatID := c.Param("chatId")
		if chatID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chatId is required"})
			return
		}

		var req sendMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		msgType := int16(0) // default to text, mapped in worker
		if req.MsgType != nil {
			msgType = *req.MsgType
		}

		payload := task.SendMessageTaskPayload{
			ConversationID: chatID,
			SenderID:       req.SenderID,
			Body:           req.Body,
			MsgType:        msgType,
			AttachmentURL:  req.AttachmentURL,
			AttachmentMeta: req.AttachmentMeta,
			DedupeKey:      req.DedupeKey,
		}
		b, err := json.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode task payload"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		// Enqueue task; best-effort options
		opts := queueport.EnqueueOption{Queue: "chat", MaxRetry: 20}
		id, err := h.Q.Enqueue(ctx, queueport.Task{Type: task.SendMessageTaskType, Payload: b}, opts)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to enqueue message"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"status":    "queued",
			"task_id":   id,
			"chat_id":   chatID,
			"sender_id": req.SenderID,
		})
	}
}
