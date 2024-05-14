-- name: CreateUser :exec
INSERT INTO users (id, email, password_hash)
VALUES (@id::UUID, @email, @password_hash);

-- name: FindUserByID :one
SELECT * FROM users WHERE id = @id::UUID LIMIT 1;

-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = @email LIMIT 1;

-- TODO: Is it possible to link a user to a mismatching user account?
-- name: LinkUserToSSOProvider :exec
INSERT INTO sso_provider_users (id, user_id, provider_name, provider_user_id)
VALUES
  (@id::UUID, @user_id::UUID, @provider_name, @provider_user_id)
RETURNING id;

-- name: FindUserBySSOProvider :one
SELECT user
FROM sso_provider_users
WHERE provider_user_id = @provider_user_id AND provider_name = @provider_name
LIMIT 1;

-- name: SetUserPasswordByID :exec
UPDATE users
SET password_hash = @password_hash
WHERE id = @id;

-- name: UpsertPasswordResetToken :one
WITH
  token AS (
    INSERT INTO password_reset_tokens (id, user_id, token, expires_at, is_used)
    VALUES
      (@id::UUID, @user_id, @token, @expires_at, FALSE)
    ON CONFLICT (user_id) DO UPDATE
      SET expires_at = greatest(
        excluded.expires_at,
        password_reset_tokens.expires_at
      )
    RETURNING token, id, expires_at
  )
SELECT *
FROM token;

-- name: FindPasswordResetToken :one
SELECT *
FROM password_reset_tokens
WHERE token = @token
LIMIT 1 AND expires_at > now();

-- name: MarkPasswordResetTokenAsUsed :exec
UPDATE password_reset_tokens
SET is_used = TRUE
WHERE token = @token;

-- name: DeleteExpiredPasswordResetTokens :exec
DELETE FROM password_reset_tokens WHERE expires_at < now() RETURNING id;
