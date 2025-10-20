package v1

import (
	qport "go-chatty/internal/infrastructure/queue/port"
	httpHandler "go-chatty/internal/pkg/chat/presentation/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterRoutes mounts all version 1 API routes under /api/v1
func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, client qport.Client) {
	v1 := r.Group("/api/v1")
	// Pass the DB connection and queue client down to the HTTP layer
	httpHandler.RegisterRoutes(v1, pool, client)
}
