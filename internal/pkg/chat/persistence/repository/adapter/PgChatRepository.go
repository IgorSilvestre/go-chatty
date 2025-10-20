package adapter

import (
	"context"
	"errors"
	chat "go-chatty/internal/pkg/chat/application/domain"
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

func (r *PgChatRepository) CreateConversation(ctx context.Context, c chat.Conversation) (string, error) {
	if r == nil || r.pool == nil {
		return "", errors.New("PgChatRepository: nil pool")
	}
	var id string
	err := r.pool.QueryRow(ctx,
		"INSERT INTO chat.conversation (created_at, tenant_id) VALUES ($1, NULLIF($2, '')::uuid) RETURNING id::text",
		c.CreatedAt, c.TenantID,
	).Scan(&id)
	return id, err
}

func (r *PgChatRepository) AddParticipant(ctx context.Context, p chat.Participant) error {
	if r == nil || r.pool == nil {
		return errors.New("PgChatRepository: nil pool")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO chat.participant (conversation_id, user_id, role, last_read_msg, muted_until)
		VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5)
		ON CONFLICT (conversation_id, user_id)
		DO UPDATE SET role = EXCLUDED.role,
		              last_read_msg = EXCLUDED.last_read_msg,
		              muted_until = EXCLUDED.muted_until
	`, p.ConversationID, p.UserID, p.Role, p.LastReadMsg, p.MutedUntil)
	return err
}

func (r *PgChatRepository) SaveMessage(ctx context.Context, m chat.Message) (string, error) {
	if r == nil || r.pool == nil {
		return "", errors.New("PgChatRepository: nil pool")
	}
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO chat.message (
			conversation_id, sender_id, created_at, body, msg_type, attachment_url, attachment_meta, dedupe_key
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, COALESCE($7::json, NULL), $8)
		RETURNING id::text
	`, m.ConversationID, m.SenderID, m.CreatedAt, m.Body, m.MsgType, m.AttachmentURL, m.AttachmentMeta, m.DedupeKey).Scan(&id)
	return id, err
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
		SELECT id::text, conversation_id::text, sender_id::text, created_at, body, msg_type, attachment_url, attachment_meta, dedupe_key
		FROM chat.message
		WHERE conversation_id = $1::uuid
		ORDER BY created_at DESC
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
		UPDATE chat.participant
		SET last_read_msg = $3::uuid
		WHERE conversation_id = $1::uuid AND user_id = $2::uuid
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
		UPDATE chat.participant
		SET muted_until = $3
		WHERE conversation_id = $1::uuid AND user_id = $2::uuid
	`, conversationID, userID, mutedUntil)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
