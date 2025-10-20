package v1

import (
	httpHandler "go-chatty/internal/pkg/chat/presentation/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterRoutes mounts all version 1 API routes under /api/v1
func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool) {
	v1 := r.Group("/api/v1")
	// Pass the DB connection down to the HTTP layer; controllers will compose repos/usecases
	httpHandler.RegisterRoutes(v1, pool)
}
