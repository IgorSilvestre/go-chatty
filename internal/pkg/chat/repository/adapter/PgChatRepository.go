package adapter

import (
	"context"
	"errors"
	"go-chatty/internal/pkg/chat/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgChatRepository struct {
	pool *pgxpool.Pool
}

func NewPgChatRepository(pool *pgxpool.Pool) *PgChatRepository {
	return &PgChatRepository{pool: pool}
}

func (r *PgChatRepository) CreateConversation(ctx context.Context, c chat.Conversation) error {
	if r == nil || r.pool == nil {
		return errors.New("PgChatRepository: nil pool")
	}
	_, err := r.pool.Exec(ctx,
		"INSERT INTO conversations (id, created_at, tenant_id) VALUES ($1, $2, $3)",
		c.ID, c.CreatedAt, c.TenantID,
	)
	return err
}

func (r *PgChatRepository) AddParticipant(ctx context.Context, p chat.Participant) error {
	if r == nil || r.pool == nil {
		return errors.New("PgChatRepository: nil pool")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO participants (conversation_id, user_id, role, last_read_msg, muted_until)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (conversation_id, user_id)
		DO UPDATE SET role = EXCLUDED.role,
		              last_read_msg = EXCLUDED.last_read_msg,
		              muted_until = EXCLUDED.muted_until
	`, p.ConversationID, p.UserID, p.Role, p.LastReadMsg, p.MutedUntil)
	return err
}

func (r *PgChatRepository) SaveMessage(ctx context.Context, m chat.Message) error {
	if r == nil || r.pool == nil {
		return errors.New("PgChatRepository: nil pool")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO messages (
			id, conversation_id, sender_id, created_at, body, msg_type, attachment_url, attachment_meta, dedupe_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8::json, NULL), $9)
	`, m.ID, m.ConversationID, m.SenderID, m.CreatedAt, m.Body, m.MsgType, m.AttachmentURL, m.AttachmentMeta, m.DedupeKey)
	return err
}

func (r *PgChatRepository) GetMessagesByConversation(ctx context.Context, conversationID string, limit int, offset int) ([]chat.Message, error) {
	if r == nil || r.pool == nil {
		return nil, errors.New("PgChatRepository: nil pool")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, sender_id, created_at, body, msg_type, attachment_url, attachment_meta, dedupe_key
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []chat.Message
	for rows.Next() {
		var (
			msg     chat.Message
			body    *string
			attURL  *string
			attMeta *string
			dedupe  *string
		)
		if err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.CreatedAt, &body, &msg.MsgType, &attURL, &attMeta, &dedupe); err != nil {
			return nil, err
		}
		msg.Body = body
		msg.AttachmentURL = attURL
		msg.AttachmentMeta = attMeta
		msg.DedupeKey = dedupe
		msgs = append(msgs, msg)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return msgs, nil
}

func (r *PgChatRepository) UpdateParticipantReadState(ctx context.Context, conversationID string, userID string, lastReadMsg *string) error {
	if r == nil || r.pool == nil {
		return errors.New("PgChatRepository: nil pool")
	}
	ct, err := r.pool.Exec(ctx, `
		UPDATE participants
		SET last_read_msg = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID, lastReadMsg)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PgChatRepository) SetMuteUntil(ctx context.Context, conversationID string, userID string, mutedUntil *time.Time) error {
	if r == nil || r.pool == nil {
		return errors.New("PgChatRepository: nil pool")
	}
	ct, err := r.pool.Exec(ctx, `
		UPDATE participants
		SET muted_until = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID, mutedUntil)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
