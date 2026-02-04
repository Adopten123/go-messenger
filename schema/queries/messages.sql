-- name: CreateChat :one
INSERT INTO chats (name, is_group)
VALUES ($1, $2)
    RETURNING *;

-- name: AddChatMember :exec
INSERT INTO chat_members (chat_id, user_id, role)
VALUES ($1, $2, $3);

-- name: CreateMessage :one
INSERT INTO messages (chat_id, sender_id, content)
VALUES ($1, $2, $3)
    RETURNING *;

-- name: ListMessages :many
SELECT
    m.id,
    m.content,
    m.created_at,
    m.sender_id,
    u.username as sender_username
FROM messages m
         JOIN users u ON m.sender_id = u.id
WHERE m.chat_id = $1
ORDER BY m.created_at DESC
    LIMIT $2 OFFSET $3;

-- name: MarkMessagesAsRead :exec
UPDATE messages
SET is_read = TRUE
WHERE chat_id = $1 AND sender_id != $2 AND is_read = FALSE;