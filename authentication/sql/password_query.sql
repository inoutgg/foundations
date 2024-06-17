-- name: CreateUserPasswordCredential :exec
INSERT INTO user_credentials
  (id, name, user_id, user_credential_key, user_credential_secret)
VALUES
  (
    @id::UUID,
    @name,
    @user_id::UUID,
    @user_credential_key,
    @user_credential_secret
  );

-- name: UpsertPasswordCredentialByUserID :exec
WITH
  credential AS (
    INSERT INTO user_credentials
      (id, name, user_id, user_credential_key, user_credential_secret)
    VALUES
      (
        @id::UUID,
        'password',
        @user_id::UUID,
        @user_credential_key,
        @user_credential_secret
      )
    ON CONFLICT (name, user_credential_key) DO UPDATE
      SET user_credential_secret = @user_credential_secret
    RETURNING id
  )
SELECT *
FROM credential;

-- name: FindUserWithPasswordCredentialByEmail :one
WITH
  credential AS (
    SELECT user_credential_key, user_credential_secret, user_id
    FROM user_credentials
    WHERE name = 'password' AND user_credential_key = @email
  ),
  "user" AS (
    SELECT *
    FROM users
    WHERE email = @email
  )
SELECT "user".*, credential.user_credential_secret AS password_hash
FROM
  credential
  -- validate that the credential and user has the same email address.
  JOIN "user"
    ON credential.user_id = "user".id;

-- name: UpsertPasswordResetToken :one
WITH
  token AS (
    INSERT INTO password_reset_tokens (id, user_id, token, expires_at, is_used)
    VALUES
      (@id::UUID, @user_id, @token, @expires_at, FALSE)
    ON CONFLICT (user_id, is_used) DO UPDATE
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
