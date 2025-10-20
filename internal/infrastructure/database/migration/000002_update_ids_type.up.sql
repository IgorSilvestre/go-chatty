-- 000002_update_ids_type.up.sql
-- Update ID columns in chat schema from CHAR/VARCHAR to UUID
-- Strategy: use deterministic uuid v5 derived from existing string values to preserve relationships.
-- Also set gen_random_uuid() defaults for primary key ids going forward.

-- Ensure required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp"; -- for uuid_generate_v5
CREATE EXTENSION IF NOT EXISTS "pgcrypto";  -- for gen_random_uuid()

-- Use a fixed namespace UUID for deterministic v5 generation
-- Note: This constant is arbitrary but must remain unchanged to keep mapping stable across tables.
-- If you change it, all derived UUIDs would change.
-- 6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f

-- Drop dependent index to avoid type dependency issues
DROP INDEX IF EXISTS chat.idx_messages_conv_created;

-- Drop FKs and PKs prior to type changes
ALTER TABLE chat.participant DROP CONSTRAINT IF EXISTS participant_conversation_id_fkey;
ALTER TABLE chat.participant DROP CONSTRAINT IF EXISTS participant_pkey;

ALTER TABLE chat.message DROP CONSTRAINT IF EXISTS message_conversation_id_fkey;
ALTER TABLE chat.message DROP CONSTRAINT IF EXISTS message_pkey;

ALTER TABLE chat.receipt DROP CONSTRAINT IF EXISTS receipt_pkey;
ALTER TABLE chat.block DROP CONSTRAINT IF EXISTS block_pkey;
ALTER TABLE chat.conversation DROP CONSTRAINT IF EXISTS conversation_pkey;

-- Convert conversation ids
ALTER TABLE chat.conversation
  ALTER COLUMN id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, btrim(id)),
  ALTER COLUMN tenant_id TYPE uuid USING CASE WHEN tenant_id IS NULL THEN NULL ELSE uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, tenant_id::text) END;

-- Convert participant ids
ALTER TABLE chat.participant
  ALTER COLUMN conversation_id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, btrim(conversation_id)),
  ALTER COLUMN user_id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, user_id::text),
  ALTER COLUMN last_read_msg TYPE uuid USING CASE WHEN last_read_msg IS NULL THEN NULL ELSE uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, btrim(last_read_msg)) END;

-- Convert message ids
ALTER TABLE chat.message
  ALTER COLUMN id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, btrim(id)),
  ALTER COLUMN conversation_id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, btrim(conversation_id)),
  ALTER COLUMN sender_id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, sender_id::text);

-- Convert receipt ids
ALTER TABLE chat.receipt
  ALTER COLUMN message_id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, btrim(message_id)),
  ALTER COLUMN user_id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, user_id::text);

-- Convert block ids
ALTER TABLE chat.block
  ALTER COLUMN blocker_id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, blocker_id::text),
  ALTER COLUMN blocked_id TYPE uuid USING uuid_generate_v5('6f2a8bf8-6b92-4c65-90a7-0b6f5c4e8b0f'::uuid, blocked_id::text);

-- Defaults for PKs to ease inserts going forward
ALTER TABLE chat.conversation ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE chat.message ALTER COLUMN id SET DEFAULT gen_random_uuid();

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
