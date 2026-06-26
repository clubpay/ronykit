Use this when a table is expected to grow continuously (events, logs, ledger entries, audit trails, metrics, messages). Partitioning keeps indexes small, makes retention cheap (drop old partitions instead of mass `DELETE`), and improves query plans when filters include the partition key.

Read this resource during SRS/SDD when NFRs mention scale, retention, or high write volume. If partitioning applies, document the strategy in SDD §6.2 and ship automated maintenance — do not leave partition creation as a manual ops step.

Scope: this resource covers **PostgreSQL declarative partitioning** (the RonyKIT default datastore). MySQL `PARTITION BY` exists but differs in syntax and limits (no native cross-partition FKs, different maintenance) — if a feature requires MySQL, adapt the strategy table and confirm in SDD.

## When to partition

Partition when **most** of these are true:

- Row count is projected to exceed ~10–50M within the retention window, or write rate is high enough that index/vacuum cost is a concern.
- Queries routinely filter (or can be rewritten to filter) on a stable partition key: `created_at`, `event_date`, `tenant_id`, `region`, etc.
- Old data can be dropped or archived by key range without row-by-row deletes.
- Retention is defined (e.g. keep 24 months, archive after 12).

Skip partitioning when the table stays small, joins/updates span arbitrary keys, or there is no natural partition key.

## Choose a strategy

| Pattern | Partition key | Good for | Granularity guidance |
|---------|---------------|----------|----------------------|
| **Time range** | `TIMESTAMPTZ` / `DATE` | Append-mostly facts: events, audit, notifications, ledger lines | See [Time-based granularity](#time-based-granularity) |
| **Range (non-time)** | Numeric ID, amount bucket | Monotonic IDs, sharded numeric ranges | Size partitions for ~1–10M rows each |
| **List** | `tenant_id`, `region`, `status` | Strong isolation per tenant/region; moderate cardinality | Avoid high-cardinality lists (thousands of tenants → use hash or time) |
| **Hash** | `tenant_id`, `user_id` | Even spread when range/list is impractical | Fixed bucket count (e.g. 16/32); hard to drop by age — pair with time if retention matters |

**Default for growing append-only data:** PostgreSQL **declarative RANGE** on a timestamp column.

### Time-based granularity

Pick interval from query patterns, retention, and partition count (aim for **tens to low hundreds** of live partitions, not thousands).

| Strategy | Interval | Typical use | Partition name example |
|----------|----------|-------------|------------------------|
| **Daily** | 1 day | Very high ingest (>1M rows/day), short retention (≤90 days) | `events_2026_06_26` |
| **Monthly** | 1 month | Default for most SaaS event/audit tables | `events_2026_06` |
| **Quarterly** | 3 months | Lower volume, long retention (years), fewer admin operations | `events_2026_q2` |
| **Yearly** | 1 year | Archival reference data, compliance snapshots | `events_2026` |

**Monthly** is the default unless SDD justifies daily (extreme volume) or quarterly/yearly (low volume + long retention).

Always create partitions **ahead of time** (see [Automated maintenance](#automated-maintenance)).

## PostgreSQL layout (declarative partitioning)

- Use native `PARTITION BY RANGE` (or `LIST` / `HASH` when the strategy table above says so).
- **Primary key and unique constraints must include the partition key** (PostgreSQL requirement).
- Prefer `TIMESTAMPTZ` stored in UTC; partition bounds in UTC.
- Parent table holds schema; physical rows live in child partitions.
- Indexes on the parent propagate to partitions — define once on the parent.
- Inserts/updates that don't match any partition fail — this is why ahead-of-time creation matters.
- Creating/dropping a partition briefly takes a lock on the parent. Keep maintenance off peak hours; use `DETACH PARTITION CONCURRENTLY` before `DROP` if you must avoid blocking reads during retention cleanup.
- Always pair the `.up.sql` migration with a `.down.sql` that drops the partitioned table (children cascade) and any maintenance function.

### Migration sketch (new partitioned table)

Number migrations sequentially in `data/db/migrations/`:

```sql
-- 00N_create_events.up.sql
CREATE TABLE events (
    id          BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at  TIMESTAMPTZ NOT NULL,
    tenant_id   UUID NOT NULL,
    payload     JSONB NOT NULL,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE INDEX events_tenant_created_idx ON events (tenant_id, created_at DESC);

-- Bootstrap current and next intervals (monthly example)
CREATE TABLE events_2026_05 PARTITION OF events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE events_2026_06 PARTITION OF events
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE events_2026_07 PARTITION OF events
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
```

### Converting an existing heap table

Only when SDD explicitly plans it (downtime or dual-write window):

1. Create partitioned parent + child partitions for the target strategy.
2. Backfill historical ranges into matching partitions (`INSERT INTO events SELECT …` with time bounds, or `ATTACH PARTITION` after creating matching child tables).
3. Swap names or cut over reads/writes; drop old heap.
4. Document the cutover and rollback plan in SDD §6.2 (schema/migrations) and record residual risks in §10 (open issues).

Do not silently partition a live table without SDD approval and a migration plan.

## sqlc and repo conventions

- Point sqlc schema at migrations as usual (`internal/repo/v0/sqlc.yml`).
- Query the **parent** table name in `data/db/queries/*.sql`; PostgreSQL routes to partitions automatically.
- **Always filter on the partition key** in hot-path queries when possible (`WHERE created_at >= @from AND created_at < @to`). Document required filters in SDD §6.3.
- For cross-partition aggregates, keep time bounds explicit in the API/app layer.
- Integration tests: cover insert/read in the current partition and verify queries with time bounds (`x/testkit` + Gnomock Postgres).

## Automated maintenance

Partitioning without automation will break inserts when the clock crosses an uncovered range. Ship **both**:

1. **SQL routine** — idempotent function(s) in migrations that create future partitions and detach/drop expired ones.
2. **Scheduler** — recurring job that executes the routine.

### SQL maintenance routine (monthly example)

Add to migrations (adjust names/granularity per SDD):

```sql
-- 00N_events_partition_maintenance.up.sql
CREATE OR REPLACE FUNCTION maintain_events_partitions(
    p_ahead_months int DEFAULT 3,
    p_retain_months int DEFAULT 24
) RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    start_month date := date_trunc('month', now() AT TIME ZONE 'UTC')::date;
    i int;
    part_from date;
    part_to date;
    part_name text;
    retain_before date := (start_month - make_interval(months => p_retain_months))::date;
    r record;
BEGIN
    -- Create ahead partitions
    FOR i IN 0..p_ahead_months LOOP
        part_from := (start_month + make_interval(months => i))::date;
        part_to := (start_month + make_interval(months => i + 1))::date;
        part_name := format('events_%s', to_char(part_from, 'YYYY_MM'));
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF events FOR VALUES FROM (%L) TO (%L)',
            part_name, part_from, part_to
        );
    END LOOP;

    -- Drop partitions older than retention (detach optional if archiving first)
    FOR r IN
        SELECT c.relname AS partition_name
        FROM pg_inherits i
        JOIN pg_class c ON c.oid = i.inhrelid
        JOIN pg_class p ON p.oid = i.inhparent
        WHERE p.relname = 'events'
          AND c.relname ~ '^events_[0-9]{4}_[0-9]{2}$'
    LOOP
        part_from := to_date(substring(r.partition_name from 'events_(.*)$'), 'YYYY_MM');
        IF part_from < retain_before THEN
            EXECUTE format('DROP TABLE IF EXISTS %I', r.partition_name);
        END IF;
    END LOOP;
END;
$$;
```

**Quarterly variant:** iterate with `make_interval(months => i * 3)`, name partitions `events_YYYY_qN` (`q1` = Jan–Mar), and match the regex/drop logic to that naming scheme.

Keep parameters (`ahead`, `retain`, interval) in `internal/settings/settings.go` so ops can tune without code changes.

### sqlc query for maintenance

```sql
-- name: MaintainEventsPartitions :exec
SELECT maintain_events_partitions(@ahead_months, @retain_months);
```

Wire a thin repo method; the app layer should not embed DDL strings.

### Scheduler options (pick one in SDD)

| Approach | When to use | RonyKIT wiring |
|----------|-------------|----------------|
| **In-process goroutine (default for most services)** | No Temporal, single owning service, maintenance is a cheap idempotent call | Run once on startup, then on an interval via an fx lifecycle hook in `module.go`. Logs with `logkit`; stops on `OnStop`. See sketch below. |
| **`pg_cron`** | DB-managed scheduling preferred, infra team manages extensions, runs even if the app is down | Migration enables the extension; `SELECT cron.schedule('maintain-events-partitions', '0 2 * * *', $$SELECT maintain_events_partitions()$$);` Document in SDD/devops — not Go code. |
| **`flow` schedule** | Service already uses `flow`; you want durable history, retries, and visibility | Calendar/interval schedule whose activity calls repo `MaintainEventsPartitions`. Read `architecture/flow-workflows`, `characteristics/workflow`. Use `ScheduleOverlapPolicySkip`. |
| **External cron / k8s CronJob** | Ops-owned, language-agnostic | Script or `psql` job calling the same SQL function. |

**Default:** the **in-process goroutine** for a service that owns the table and isn't already on `flow`. Use **`pg_cron`** when scheduling should live in the DB (survives app restarts/downtime). Use a **`flow` schedule** only when the module already wires `flow`. Pick exactly one owner per table and record it in SDD §8 — never run two schedulers against the same table.

Minimum cadence: **daily** check (the function is cheap and idempotent). For daily partitions, run **hourly** near day boundaries. Because the function is idempotent, a redundant run is harmless — running once on startup guarantees partitions exist even after downtime.

#### In-process goroutine sketch (default)

Wire a startup + interval runner in `module.go` using the same `fx.Lifecycle` pattern the service template already uses. Run once on `OnStart`, then on a ticker; cancel cleanly on `OnStop`. Keep the DDL in the repo method (`MaintainEventsPartitions`), not inline.

```go
// module.go — provide a maintenance runner bound to the fx lifecycle
fx.Invoke(func(lc fx.Lifecycle, repo repo.EventRepository, set *settings.Settings, l *logkit.Logger) {
    ctx, cancel := context.WithCancel(context.Background())
    var wg sync.WaitGroup

    run := func() {
        if err := repo.MaintainEventsPartitions(ctx, set.PartitionAhead, set.PartitionRetain); err != nil {
            l.Error("partition maintenance failed", logkit.Error(err))
        }
    }

    lc.Append(fx.Hook{
        OnStart: func(context.Context) error {
            run() // ensure partitions exist immediately, even after downtime
            wg.Add(1)
            go func() {
                defer wg.Done()
                t := time.NewTicker(set.PartitionMaintainEvery) // e.g. 24h
                defer t.Stop()
                for {
                    select {
                    case <-ctx.Done():
                        return
                    case <-t.C:
                        run()
                    }
                }
            }()
            return nil
        },
        OnStop: func(context.Context) error {
            cancel()
            wg.Wait()
            return nil
        },
    })
})
```

Notes:

- Keep `PartitionAhead`, `PartitionRetain`, and `PartitionMaintainEvery` in `internal/settings/settings.go` so ops can tune them.
- The startup run is the key reliability property: if the service was down across a month boundary, the missing partition is created before the first insert.
- In multi-replica deployments every replica will run it; that's safe (idempotent), but if you want a single runner use `pg_cron` or a `flow` schedule instead.

#### `flow` wiring sketch (when already using `flow`)

Keep orchestration in the workflow file and the DB call in an activity (read `architecture/flow-workflows`, `characteristics/workflow`). Register a recurring schedule at startup; the activity calls the repo method that runs the maintenance function.

```go
// internal/app/partition_maintenance_workflow.go — orchestration only
func MaintainPartitionsWorkflow(ctx flow.Context, _ flow.EMPTY) (flow.EMPTY, error) {
    return ctx.ExecuteActivity(MaintainPartitionsActivity, flow.EMPTY{})
}

// internal/app/workflow_activities.go — side effects via app/repo deps
func MaintainPartitionsActivity(ctx flow.Context, _ flow.EMPTY) (flow.EMPTY, error) {
    app := ctx.S() // typed app state injected via InitWithState
    return flow.EMPTY{}, app.eventRepo.MaintainEventsPartitions(ctx, settings.PartitionAhead, settings.PartitionRetain)
}

// at service start: create an idempotent calendar schedule (overlap = skip)
_, err := sdk.CreateSchedule(ctx, flow.CreateScheduleRequest{
    ID:     "maintain-events-partitions",
    Action: flow.ScheduleAction{WorkflowName: "MaintainPartitionsWorkflow"},
    Spec:   flow.ScheduleSpec{Calendars: []flow.ScheduleCalendarSpec{{Hour: 2, Minute: 0}}},
    OverlapPolicy: flow.ScheduleOverlapPolicySkip,
})
```

(Names follow the workflow file conventions in `characteristics/workflow`; treat snippets as a shape, not a drop-in.)

### Observability

- Log partition names created/dropped and counts from the maintenance activity.
- Emit a metric (via `meterkit`) for last successful maintenance timestamp.
- Alert if insert errors mention missing partition (should not happen if automation works).

## SDD checklist (§6 Persistence)

When partitioning applies, SDD must include:

| Item | Example |
|------|---------|
| Partition key + strategy | `created_at`, RANGE monthly |
| Granularity rationale | Monthly: ~500k rows/month, 24-month retention |
| Naming convention | `events_YYYY_MM` |
| Ahead window | Create 3 months ahead |
| Retention / archival | Drop after 24 months; optional S3 export before drop |
| Maintenance function | `maintain_events_partitions(ahead, retain)` |
| Scheduler + owner | in-process goroutine (startup + 24h) / `pg_cron` / `flow` — one owner only |
| Query rules | All list APIs require `from`/`to` on `created_at` |
| Migration / backfill plan | New table vs convert existing |

## Agent workflow summary

1. During SRS/SDD, estimate growth and retention; if partitioning is warranted, read this resource.
2. Choose strategy (default: time RANGE **monthly**; quarterly for low volume + long retention).
3. Add partitioned parent + initial partitions in migrations; include maintenance function in a follow-up migration.
4. Add sqlc query + repo port method for maintenance; wire scheduled execution per SDD.
5. Ensure handlers/app enforce partition-key filters on hot paths.
6. Add integration tests for time-bounded queries and maintenance idempotency.
