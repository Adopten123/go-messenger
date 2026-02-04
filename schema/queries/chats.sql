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