-- name: GetChatMembers :many
SELECT user_id
FROM chat_members
WHERE chat_id = $1;

-- name: IsChatMember :one
SELECT EXISTS (
    SELECT 1
    FROM chat_members
    WHERE chat_id = $1 AND user_id = $2
);

-- name: ListUserChats :many
SELECT
    c.id,
    c.name,
    c.is_group,
    c.created_at
FROM chats c
         JOIN chat_members cm ON c.id = cm.chat_id
WHERE cm.user_id = $1
ORDER BY c.created_at DESC;