package http

import (
	"go-chatty/internal/pkg/chat/presentation/controller"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterRoutes registers chat-related HTTP endpoints under the given router group
// It constructs per-endpoint controllers and binds them directly to routes.
func RegisterRoutes(g *gin.RouterGroup, pool *pgxpool.Pool) {
	createCtl := controller.NewCreateChatController(pool)
	sendMsgCtl := controller.NewSendMessageController(pool)

	// POST /api/v1/chat -> create a chat
	g.POST("/chat", createCtl.Handle())

	// POST /api/v1/chat/:chatId -> send a message into a chat
	g.POST("/chat/:chatId", sendMsgCtl.Handle())
}
