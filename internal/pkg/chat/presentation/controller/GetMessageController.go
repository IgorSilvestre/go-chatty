package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"go-chatty/internal/pkg/chat/application/usecase"
	"go-chatty/internal/pkg/chat/persistence/repository/adapter"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetMessageController handles fetching messages by chat ID (one controller per endpoint)
type GetMessageController struct {
	UC *usecase.GetMessageUseCase
}

func NewGetMessageController(pool *pgxpool.Pool) *GetMessageController {
	repo := adapter.NewPgChatRepository(pool)
	uc := usecase.NewGetMessageUseCase(repo)
	return &GetMessageController{UC: uc}
}

func (h *GetMessageController) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		chatID := c.Param("chatId")
		if chatID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chatId is required"})
			return
		}

		// Defaults
		limit := 50
		offset := 0

		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if v := c.Query("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}

		in := usecase.GetMessageInput{ConversationID: chatID, Limit: limit, Offset: offset}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		msgs, err := h.UC.Execute(ctx, in)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrPersistence) {
				status = http.StatusInternalServerError
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		// Serialize messages as-is; field names kept explicit for clarity
		out := make([]gin.H, 0, len(msgs))
		for _, m := range msgs {
			out = append(out, gin.H{
				"id":              m.ID,
				"conversation_id": m.ConversationID,
				"sender_id":       m.SenderID,
				"created_at":      m.CreatedAt,
				"body":            m.Body,
				"msg_type":        m.MsgType,
				"attachment_url":  m.AttachmentURL,
				"attachment_meta": m.AttachmentMeta,
				"dedupe_key":      m.DedupeKey,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"messages": out,
			"limit":    limit,
			"offset":   offset,
			"count":    len(out),
		})
	}
}
