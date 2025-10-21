package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go-chatty/internal/infrastructure/realtime"
	chat "go-chatty/internal/pkg/chat/application/domain"
	"go-chatty/internal/pkg/chat/application/usecase"
	repoAdapter "go-chatty/internal/pkg/chat/persistence/repository/adapter"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChatSocketController handles the websocket endpoint for realtime chat traffic.
type ChatSocketController struct {
	router          *realtime.Router
	sendMessageUC   *usecase.SendMessageUseCase
	joinRoomUC      *usecase.JoinConversationUseCase
	listMembersUC   *usecase.ListParticipantsUseCase
	inflightTimeout time.Duration
}

func NewChatSocketController(pool *pgxpool.Pool, router *realtime.Router) *ChatSocketController {
	repo := repoAdapter.NewPgChatRepository(pool)
	return &ChatSocketController{
		router:          router,
		sendMessageUC:   usecase.NewSendMessageUseCase(repo),
		joinRoomUC:      usecase.NewJoinConversationUseCase(repo),
		listMembersUC:   usecase.NewListParticipantsUseCase(repo),
		inflightTimeout: 5 * time.Second,
	}
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now; plug a proper checker when auth is added.
		return true
	},
}

type inboundFrame struct {
	Type           string  `json:"type"`
	ConversationID string  `json:"conversation_id,omitempty"`
	Body           *string `json:"body,omitempty"`
	MsgType        *int16  `json:"msg_type,omitempty"`
	AttachmentURL  *string `json:"attachment_url,omitempty"`
	AttachmentMeta *string `json:"attachment_meta,omitempty"`
	DedupeKey      *string `json:"dedupe_key,omitempty"`
}

type errorFrame struct {
	Type  string `json:"type"`
	Code  string `json:"code"`
	Error string `json:"error"`
}

type ackFrame struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id,omitempty"`
}

type outboundMessage struct {
	Type           string         `json:"type"`
	ConversationID string         `json:"conversation_id"`
	Message        messagePayload `json:"message"`
}

type messagePayload struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	CreatedAt      time.Time `json:"created_at"`
	Body           *string   `json:"body,omitempty"`
	MsgType        int16     `json:"msg_type"`
	AttachmentURL  *string   `json:"attachment_url,omitempty"`
	AttachmentMeta *string   `json:"attachment_meta,omitempty"`
	DedupeKey      *string   `json:"dedupe_key,omitempty"`
}

const defaultReadTimeout = 60 * time.Second

// Handle upgrades HTTP connections to websocket and processes frames until the client disconnects.
func (ctl *ChatSocketController) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		ws, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			// Upgrade already wrote the response; just log and return.
			return
		}

		conn := realtime.NewConnection(userID, ws)
		ctl.router.Attach(conn)
		defer func() {
			ctl.router.Detach(conn)
			conn.Close(websocket.CloseNormalClosure, "session closed")
		}()

		ws.SetReadLimit(1 << 20) // 1MB payload cap
		_ = ws.SetReadDeadline(time.Now().Add(defaultReadTimeout))
		ws.SetPongHandler(func(string) error {
			return ws.SetReadDeadline(time.Now().Add(defaultReadTimeout))
		})

		handshakeAck := ackFrame{Type: "connected"}
		if payload, err := json.Marshal(handshakeAck); err == nil {
			_ = conn.Send(payload)
		}

		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) ||
					errors.Is(err, websocket.ErrCloseSent) {
					return
				}
				ctl.replyError(conn, "read_error", err.Error())
				return
			}

			var frame inboundFrame
			if err := json.Unmarshal(data, &frame); err != nil {
				ctl.replyError(conn, "bad_request", "invalid payload")
				continue
			}

			switch frame.Type {
			case "join":
				ctl.handleJoin(c, conn, frame)
			case "leave":
				ctl.handleLeave(conn, frame)
			case "message":
				ctl.handleMessage(c, conn, userID, frame)
			default:
				ctl.replyError(conn, "unsupported_type", "unknown frame type")
			}
		}
	}
}

func (ctl *ChatSocketController) handleJoin(c *gin.Context, conn *realtime.Connection, frame inboundFrame) {
	if frame.ConversationID == "" {
		ctl.replyError(conn, "bad_request", "conversation_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), ctl.inflightTimeout)
	defer cancel()

	err := ctl.joinRoomUC.Execute(ctx, usecase.JoinConversationInput{
		ConversationID: frame.ConversationID,
		UserID:         conn.UserID,
	})
	if err != nil {
		ctl.handleUseCaseError(conn, err)
		return
	}

	ctl.router.Join(frame.ConversationID, conn)

	ack := ackFrame{Type: "joined", ConversationID: frame.ConversationID}
	if payload, err := json.Marshal(ack); err == nil {
		_ = conn.Send(payload)
	}
}

func (ctl *ChatSocketController) handleLeave(conn *realtime.Connection, frame inboundFrame) {
	if frame.ConversationID == "" {
		ctl.replyError(conn, "bad_request", "conversation_id is required")
		return
	}
	ctl.router.Leave(frame.ConversationID, conn)

	ack := ackFrame{Type: "left", ConversationID: frame.ConversationID}
	if payload, err := json.Marshal(ack); err == nil {
		_ = conn.Send(payload)
	}
}

func (ctl *ChatSocketController) handleMessage(c *gin.Context, conn *realtime.Connection, userID string, frame inboundFrame) {
	if frame.ConversationID == "" {
		ctl.replyError(conn, "bad_request", "conversation_id is required")
		return
	}

	msgType := chat.MessageTypeText
	if frame.MsgType != nil {
		msgType = chat.MessageType(*frame.MsgType)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), ctl.inflightTimeout)
	defer cancel()

	result, err := ctl.sendMessageUC.Execute(ctx, usecase.SendMessageInput{
		ConversationID: frame.ConversationID,
		SenderID:       userID,
		Body:           frame.Body,
		MsgType:        msgType,
		AttachmentURL:  frame.AttachmentURL,
		AttachmentMeta: frame.AttachmentMeta,
		DedupeKey:      frame.DedupeKey,
	})
	if err != nil {
		ctl.handleUseCaseError(conn, err)
		return
	}

	out := outboundMessage{
		Type:           "message",
		ConversationID: frame.ConversationID,
		Message:        toPayload(*result),
	}

	payload, err := json.Marshal(out)
	if err != nil {
		ctl.replyError(conn, "internal_error", "failed to encode message")
		return
	}

	participants, err := ctl.listParticipants(ctx, frame.ConversationID)
	if err != nil {
		ctl.handleUseCaseError(conn, err)
		return
	}

	delivered := ctl.router.Broadcast(frame.ConversationID, payload, userID)

	if !ctl.router.NotifyUser(userID, payload) {
		_ = conn.Send(payload)
	}

	ctl.forwardToPeerNodes(participants, userID, payload, delivered)
}

func (ctl *ChatSocketController) listParticipants(ctx context.Context, conversationID string) ([]string, error) {
	return ctl.listMembersUC.Execute(ctx, usecase.ListParticipantsInput{ConversationID: conversationID})
}

func (ctl *ChatSocketController) handleUseCaseError(conn *realtime.Connection, err error) {
	switch {
	case errors.Is(err, usecase.ErrPersistence):
		ctl.replyError(conn, "internal_error", "unexpected persistence error")
	case errors.Is(err, chat.ErrNotParticipant):
		ctl.replyError(conn, "forbidden", "user is not a participant in this conversation")
	default:
		ctl.replyError(conn, "bad_request", err.Error())
	}
}

func (ctl *ChatSocketController) replyError(conn *realtime.Connection, code string, message string) {
	frame := errorFrame{
		Type:  "error",
		Code:  code,
		Error: message,
	}
	if payload, err := json.Marshal(frame); err == nil {
		_ = conn.Send(payload)
	}
}

func (ctl *ChatSocketController) forwardToPeerNodes(participants []string, senderID string, payload []byte, delivered int) {
	expected := 0
	for _, id := range participants {
		if id == senderID {
			continue
		}
		expected++
	}
	if delivered >= expected {
		return
	}
	// TODO: integrate pub/sub (e.g., Redis, NATS) to deliver payload to members connected on other nodes.
	_ = payload
}

func toPayload(msg chat.Message) messagePayload {
	return messagePayload{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		CreatedAt:      msg.CreatedAt,
		Body:           msg.Body,
		MsgType:        int16(msg.MsgType),
		AttachmentURL:  msg.AttachmentURL,
		AttachmentMeta: msg.AttachmentMeta,
		DedupeKey:      msg.DedupeKey,
	}
}
