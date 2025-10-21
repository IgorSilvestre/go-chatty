package controller

import (
	"context"
	"errors"
	"go-chatty/internal/pkg/chat/application/usecase"
	"go-chatty/internal/pkg/chat/persistence/repository/adapter"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateChatController struct {
	UC *usecase.CreateChatUseCase
}

func NewCreateChatController(pool *pgxpool.Pool) *CreateChatController {
	repo := adapter.NewPgChatRepository(pool)
	uc := usecase.NewCreateChatUseCase(repo)
	return &CreateChatController{UC: uc}
}

type createChatRequest struct {
	ParticipantIDs []string `json:"participantIds"`
}

func (h *CreateChatController) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if len(req.ParticipantIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "participantIds must include at least one user id"})
			return
		}

		in := usecase.CreateChatInput{ParticipantIDs: req.ParticipantIDs}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		conv, err := h.UC.Execute(ctx, in)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrPersistence) {
				status = http.StatusInternalServerError
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":        conv.ID,
			"createdAt": conv.CreatedAt,
		})
	}
}
