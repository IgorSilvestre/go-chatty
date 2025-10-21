package http

import (
	qport "go-chatty/internal/infrastructure/queue/port"
	"go-chatty/internal/infrastructure/realtime"
	"go-chatty/internal/pkg/chat/presentation/controller"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterRoutes registers chat-related HTTP endpoints under the given router group
// It constructs per-endpoint controllers and binds them directly to routes.
func RegisterRoutes(g *gin.RouterGroup, pool *pgxpool.Pool, client qport.Client, router *realtime.Router) {
	createCtl := controller.NewCreateChatController(pool)
	sendMsgCtl := controller.NewSendMessageController(pool, client)
	getMsgCtl := controller.NewGetMessageController(pool)
	socketCtl := controller.NewChatSocketController(pool, router)

	// POST /api/v1/chat -> create a chat
	g.POST("/chat", createCtl.Handle())

	// POST /api/v1/chat/:chatId -> send a message into a chat
	g.POST("/chat/:chatId", sendMsgCtl.Handle())

	// GET /api/v1/chat/:chatId/messages -> fetch messages by chat id
	g.GET("/chat/:chatId/messages", getMsgCtl.Handle())

	// GET /api/v1/chat/ws -> websocket endpoint for realtime chat
	g.GET("/chat/ws", socketCtl.Handle())
}
