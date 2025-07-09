-- name: GetPendingMessages :many
SELECT
    id,
    content,
    recipient_phone_number,
    status,
    external_message_id,
    created_at,
    updated_at
FROM notifications.messages
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: UpdateMessageStatus :exec
UPDATE notifications.messages
SET
    status = $3,
    external_message_id = $1,
    updated_at = NOW(),
    last_failure_reason = $4
WHERE id = $2;

-- name: GetAllSentMessages :many
SELECT
    id,
    content,
    recipient_phone_number,
    status,
    external_message_id,
    created_at,
    updated_at
FROM notifications.messages
WHERE status = 'sent'
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateMessage :one
INSERT INTO notifications.messages (
    id,
    content,
    recipient_phone_number,
    status
) VALUES (
    $1, $2, $3, 'pending'
)
RETURNING id;   