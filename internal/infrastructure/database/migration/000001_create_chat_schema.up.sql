-- Chat schema migration (UP)
-- conversations: one row per 1:1 thread
CREATE SCHEMA IF NOT EXISTS chat;

CREATE TABLE IF NOT EXISTS chat.conversation (
  id            CHAR(26) PRIMARY KEY,     -- ulid
  created_at    TIMESTAMP NOT NULL,
  tenant_id     VARCHAR(64)
);

-- participants: membership + read state
CREATE TABLE IF NOT EXISTS chat.participant (
  conversation_id CHAR(26) NOT NULL,
  user_id         VARCHAR(64) NOT NULL,          -- use UUID or string representation
  role            SMALLINT NOT NULL DEFAULT 0,   -- future-proof for groups
  last_read_msg   CHAR(26),                      -- ulid of last read
  muted_until     TIMESTAMP NULL,
  PRIMARY KEY (conversation_id, user_id),
  FOREIGN KEY (conversation_id) REFERENCES chat.conversation(id) ON DELETE CASCADE
);

-- messages: immutable log
CREATE TABLE IF NOT EXISTS chat.message (
  id              CHAR(26) PRIMARY KEY,   -- ulid
  conversation_id CHAR(26) NOT NULL,
  sender_id       VARCHAR(64) NOT NULL,
  created_at      TIMESTAMP NOT NULL,
  body            TEXT,                   -- nullable if attachment-only
  msg_type        SMALLINT NOT NULL DEFAULT 0, -- 0=text, 1=image, 2=file, 3=system
  attachment_url  TEXT,
  attachment_meta JSON,                   -- JSON or JSONB where supported
  dedupe_key      VARCHAR(64),            -- for at-least-once client retries
  FOREIGN KEY (conversation_id) REFERENCES chat.conversation(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_conv_created ON chat.message(conversation_id, created_at);

-- receipts (optional if you keep last_read_msg in participants)
CREATE TABLE IF NOT EXISTS chat.receipt (
  message_id   CHAR(26) NOT NULL,
  user_id      VARCHAR(64) NOT NULL,
  status       SMALLINT NOT NULL,    -- 0=delivered,1=read
  at           TIMESTAMP NOT NULL,
  PRIMARY KEY (message_id, user_id, status)
);

-- blocks (because humans)
CREATE TABLE IF NOT EXISTS chat.block (
  blocker_id VARCHAR(64) NOT NULL,
  blocked_id VARCHAR(64) NOT NULL,
  at         TIMESTAMP NOT NULL,
  PRIMARY KEY (blocker_id, blocked_id)
);
