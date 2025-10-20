-- 000002_update_ids_type.down.sql
-- Revert ID columns in chat schema from UUID back to string-based types.
-- Note: Since original ULID/VARCHAR content cannot be recovered from UUID v5, we convert back to VARCHAR(64) to avoid data loss via truncation.

-- Drop dependent index
DROP INDEX IF EXISTS chat.idx_messages_conv_created;

-- Drop FKs and PKs prior to type changes
ALTER TABLE chat.participant DROP CONSTRAINT IF EXISTS participant_conversation_id_fkey;
ALTER TABLE chat.participant DROP CONSTRAINT IF EXISTS participant_pkey;

ALTER TABLE chat.message DROP CONSTRAINT IF EXISTS message_conversation_id_fkey;
ALTER TABLE chat.message DROP CONSTRAINT IF EXISTS message_pkey;

ALTER TABLE chat.receipt DROP CONSTRAINT IF EXISTS receipt_pkey;
ALTER TABLE chat.block DROP CONSTRAINT IF EXISTS block_pkey;
ALTER TABLE chat.conversation DROP CONSTRAINT IF EXISTS conversation_pkey;

-- Remove defaults for ids
ALTER TABLE chat.conversation ALTER COLUMN id DROP DEFAULT;
ALTER TABLE chat.message ALTER COLUMN id DROP DEFAULT;

-- Convert types back to string types (VARCHAR(64))
ALTER TABLE chat.conversation
  ALTER COLUMN id TYPE VARCHAR(64) USING id::text,
  ALTER COLUMN tenant_id TYPE VARCHAR(64) USING tenant_id::text;

ALTER TABLE chat.participant
  ALTER COLUMN conversation_id TYPE VARCHAR(64) USING conversation_id::text,
  ALTER COLUMN user_id TYPE VARCHAR(64) USING user_id::text,
  ALTER COLUMN last_read_msg TYPE VARCHAR(64) USING last_read_msg::text;

ALTER TABLE chat.message
  ALTER COLUMN id TYPE VARCHAR(64) USING id::text,
  ALTER COLUMN conversation_id TYPE VARCHAR(64) USING conversation_id::text,
  ALTER COLUMN sender_id TYPE VARCHAR(64) USING sender_id::text;

ALTER TABLE chat.receipt
  ALTER COLUMN message_id TYPE VARCHAR(64) USING message_id::text,
  ALTER COLUMN user_id TYPE VARCHAR(64) USING user_id::text;

ALTER TABLE chat.block
  ALTER COLUMN blocker_id TYPE VARCHAR(64) USING blocker_id::text,
  ALTER COLUMN blocked_id TYPE VARCHAR(64) USING blocked_id::text;

-- Recreate PKs
ALTER TABLE chat.conversation ADD CONSTRAINT conversation_pkey PRIMARY KEY (id);
ALTER TABLE chat.participant ADD CONSTRAINT participant_pkey PRIMARY KEY (conversation_id, user_id);
ALTER TABLE chat.message ADD CONSTRAINT message_pkey PRIMARY KEY (id);
ALTER TABLE chat.receipt ADD CONSTRAINT receipt_pkey PRIMARY KEY (message_id, user_id, status);
ALTER TABLE chat.block ADD CONSTRAINT block_pkey PRIMARY KEY (blocker_id, blocked_id);

-- Recreate FKs
ALTER TABLE chat.participant
  ADD CONSTRAINT participant_conversation_id_fkey FOREIGN KEY (conversation_id)
  REFERENCES chat.conversation(id) ON DELETE CASCADE;

ALTER TABLE chat.message
  ADD CONSTRAINT message_conversation_id_fkey FOREIGN KEY (conversation_id)
  REFERENCES chat.conversation(id) ON DELETE CASCADE;

-- Recreate index
CREATE INDEX IF NOT EXISTS idx_messages_conv_created ON chat.message(conversation_id, created_at);
