-- name: CreateUserSession :one
INSERT INTO user_sessions (id, user_id, expires_at)
VALUES (@id::UUID, @user_id::UUID, @expires_at)
RETURNING id;

-- name: FindUserSessionByID :one
SELECT *
FROM user_sessions
WHERE id = @id::UUID AND expires_at < NOW()
LIMIT 1;

-- name: ExpireSessionByID :one
UPDATE user_sessions SET expires_at = NOW() WHERE id = @id::UUID RETURNING id;

-- name: ExpireAllSessionsByUserID :many
UPDATE user_sessions
SET expires_at = NOW()
WHERE user_id = @user_id::UUID
RETURNING id;
