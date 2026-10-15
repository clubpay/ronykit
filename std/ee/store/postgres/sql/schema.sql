-- ee Postgres store schema. Applied by Store.Migrate; every statement is
-- idempotent. Also the schema sqlc generates internal/db from.

CREATE TABLE IF NOT EXISTS ee_states (
    kind       text        NOT NULL,
    id         text        NOT NULL,
    version    bigint      NOT NULL,
    data       jsonb       NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (kind, id)
);

CREATE TABLE IF NOT EXISTS ee_outbox (
    seq           bigserial   PRIMARY KEY,
    id            text        NOT NULL UNIQUE,
    partition_key text        NOT NULL DEFAULT '',
    data          bytea       NOT NULL,
    created_at    timestamptz NOT NULL,
    leased_until  timestamptz
);

-- Serves the per-key ordering check of ClaimOutbox.
CREATE INDEX IF NOT EXISTS ee_outbox_partition_key_seq ON ee_outbox (partition_key, seq);

-- Names of the WithMigration steps already applied.
CREATE TABLE IF NOT EXISTS ee_migrations (
    name       text        PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);
