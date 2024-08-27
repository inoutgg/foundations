-- name: AcquireLock :exec
SELECT pg_advisory_lock(@lock_num::int);

-- name: ReleaseLock :exec
SELECT pg_advisory_unlock(@lock_num::int);

-- name: FindAllExistingMigrations :many
SELECT * FROM migrations ORDER BY version;

-- name: DoesTableExist :one
SELECT COALESCE(to_regclass(@table_name), FALSE) = FALSE;
