-- name: GetState :one
SELECT kind, id, version, data
FROM ee_states
WHERE kind = $1 AND id = $2;

-- name: InsertState :execrows
INSERT INTO ee_states (kind, id, version, data)
VALUES ($1, $2, 1, $3)
ON CONFLICT (kind, id) DO NOTHING;

-- name: UpdateState :execrows
UPDATE ee_states
SET data = sqlc.arg(data), version = version + 1, updated_at = now()
WHERE kind = sqlc.arg(kind) AND id = sqlc.arg(id) AND version = sqlc.arg(version);

-- name: DeleteState :execrows
DELETE FROM ee_states
WHERE kind = $1 AND id = $2 AND version = $3;

-- name: StateExists :one
SELECT EXISTS (SELECT 1 FROM ee_states WHERE kind = $1 AND id = $2);

-- name: AppendOutbox :copyfrom
INSERT INTO ee_outbox (id, partition_key, data, created_at)
VALUES ($1, $2, $3, $4);

-- name: LockOutboxClaims :exec
-- Serializes concurrent claimers for the duration of their transaction, so
-- the per-key check below never races with another claim in flight.
SELECT pg_advisory_xact_lock(hashtext('ee_outbox_claim'));

-- name: ClaimOutbox :many
-- Leases the oldest claimable entries. An entry is skipped while an older
-- entry with the same non-empty partition key is still leased.
UPDATE ee_outbox AS o
SET leased_until = now() + make_interval(secs => sqlc.arg(lease_seconds)::float8)
WHERE o.seq IN (
    SELECT c.seq
    FROM ee_outbox AS c
    WHERE (c.leased_until IS NULL OR c.leased_until <= now())
      AND (c.partition_key = '' OR NOT EXISTS (
          SELECT 1
          FROM ee_outbox AS p
          WHERE p.partition_key = c.partition_key
            AND p.seq < c.seq
            AND p.leased_until > now()
      ))
    ORDER BY c.seq
    LIMIT sqlc.arg(max_entries)::int
)
RETURNING o.seq, o.id, o.partition_key, o.data, o.created_at;

-- name: AckOutbox :exec
DELETE FROM ee_outbox
WHERE id = ANY(sqlc.arg(ids)::text[]);

-- name: MigrationApplied :one
SELECT EXISTS (SELECT 1 FROM ee_migrations WHERE name = $1);

-- name: RecordMigration :exec
INSERT INTO ee_migrations (name) VALUES ($1);

-- name: LockMigrations :exec
SELECT pg_advisory_xact_lock(hashtext('ee_migrations'));
