-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS notifications;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE notifications.message_status AS ENUM (
    'pending',
    'sending',
    'sent',
    'failed'
);

CREATE TABLE notifications.messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    recipient_phone_number VARCHAR(20) NOT NULL,
    status notifications.message_status NOT NULL DEFAULT 'pending',
    external_message_id VARCHAR(255) NULL,
    last_failure_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    CONSTRAINT content_length_check CHECK (char_length(content) <= 250)
);

CREATE INDEX idx_pending_messages ON notifications.messages (created_at) WHERE status = 'pending';
CREATE INDEX idx_messages_status ON notifications.messages (status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notifications.messages;
DROP TYPE IF EXISTS notifications.message_status;
-- +goose StatementEnd
