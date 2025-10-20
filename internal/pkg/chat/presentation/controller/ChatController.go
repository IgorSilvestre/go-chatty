package controller

import (
	"context"
	"errors"
	chat "go-chatty/internal/pkg/chat/application/domain"
	"go-chatty/internal/pkg/chat/application/usecase"
	"go-chatty/internal/pkg/chat/persistence/repository/adapter"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SendMessageController handles the send-message endpoint only (one controller per endpoint)
type SendMessageController struct {
	UC *usecase.SendMessageUseCase
}

func NewSendMessageController(pool *pgxpool.Pool) *SendMessageController {
	repo := adapter.NewPgChatRepository(pool)
	uc := usecase.NewSendMessageUseCase(repo)
	return &SendMessageController{UC: uc}
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

// Handle returns a gin handler that handles sending a message to a conversation
func (h *SendMessageController) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		conversationID := c.Param("conversationId")
		if conversationID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "conversationId is required"})
			return
		}

		var req sendMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		msgType := chat.MessageTypeText
		if req.MsgType != nil {
			msgType = chat.MessageType(*req.MsgType)
		}

		in := usecase.SendMessageInput{
			ConversationID: conversationID,
			SenderID:       req.SenderID,
			Body:           req.Body,
			MsgType:        msgType,
			AttachmentURL:  req.AttachmentURL,
			AttachmentMeta: req.AttachmentMeta,
			DedupeKey:      req.DedupeKey,
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		msg, err := h.UC.Execute(ctx, in)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrPersistence) {
				status = http.StatusInternalServerError
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":              msg.ID,
			"conversation_id": msg.ConversationID,
			"sender_id":       msg.SenderID,
			"created_at":      msg.CreatedAt,
			"body":            msg.Body,
			"msg_type":        msg.MsgType,
			"attachment_url":  msg.AttachmentURL,
			"attachment_meta": msg.AttachmentMeta,
			"dedupe_key":      msg.DedupeKey,
		})
	}
}
