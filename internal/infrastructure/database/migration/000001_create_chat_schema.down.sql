-- Chat schema migration (DOWN)
-- Drop objects in reverse order of creation to satisfy dependencies
DROP TABLE IF EXISTS chat.block;
DROP TABLE IF EXISTS chat.receipt;
DROP INDEX IF EXISTS chat.idx_messages_conv_created;
DROP TABLE IF EXISTS chat.message;
DROP TABLE IF EXISTS chat.participant;
DROP TABLE IF EXISTS chat.conversation;
