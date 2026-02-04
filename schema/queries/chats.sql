-- name: GetChatMembers :many
SELECT user_id
FROM chat_members
WHERE chat_id = $1;