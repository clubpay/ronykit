# SDD template (IEEE 1016–2009 style, RonyKIT mapping)

Use this outline when writing `docs/design/<feature_name>-sdd.md`. The SDD must trace back to SRS requirement IDs (FR-*, NFR-*). Map every design element to the scaffolded module layout.

## Required frontmatter (approval gate)

The file **must** begin with this YAML frontmatter. The `scaffold_feature` tool refuses to run until this document has `status: approved`:

```yaml
---
feature: <feature_name>
document: SDD
status: draft   # set to "approved" ONLY after the user reviews and approves
---
```

Write `status: draft` when you create the document. **Never** set `status: approved` yourself — only the user approves it (by editing the file or explicitly telling you to). This is enforced by `scaffold_feature`, not just convention.

## Document control

| Field             | Value                               |
|-------------------|-------------------------------------|
| Feature           | `<feature_name>`                    |
| Go module package | `<feature_name>mod`                 |
| SRS reference     | `docs/design/<feature_name>-srs.md` |
| Version           | 1.0                                 |
| Status            | Draft / Review / Approved           |

## 1. Design overview

### 1.1 Purpose

[What this design document specifies.]

### 1.2 Scope

[Modules and boundaries covered by this design.]

### 1.3 Design constraints

[RonyKIT conventions: thin handlers, repo ports, x/ toolkit packages, error codes.]

## 2. Architecture

### 2.1 Module structure

```
feature/<feature_name>/
  service.go, module.go, migration.go
  api/service.go, api/api_<domain>.go
  internal/app/, internal/domain/, internal/repo/, internal/settings/
```

### 2.2 Component diagram

[ASCII or mermaid: Client → Gateway → handlers → app → repo → DB.]

### 2.3 Requirement traceability

| SRS ID | Design element | Location           |
|--------|----------------|--------------------|
| FR-001 |                | `internal/app/...` |

## 3. API design (`api/`)

For each operation from the SRS, specify:

| Operation | Route | Handler | Input fields | Output fields | Domain errors |
|-----------|-------|---------|--------------|---------------|---------------|
|           |       |         |              |               |               |

Include `Desc()` registration notes (`rony.WithUnary`, metadata, auth middleware if any).

## 4. Domain design (`internal/domain/`)

### 4.1 Types

| Type | Fields | Notes |
|------|--------|-------|

### 4.2 Domain errors

| Code | errs.ErrCode | When |
|------|--------------|------|

## 5. Application layer (`internal/app/`)

| Use case | Method | Inputs | Returns | SRS IDs |
|----------|--------|--------|---------|---------|
|          |        |        |         |         |

Business rules, validation beyond API boundary, idempotency keys if applicable.

## 6. Persistence (`internal/repo/`)

### 6.1 Port interfaces (`port.go`)

| Interface | Methods | Domain types |
|-----------|---------|--------------|

### 6.2 Schema / migrations

| Table | Columns | Indexes | Notes |
|-------|---------|---------|-------|

### 6.3 sqlc queries (outline)

[List query names and purpose.]

## 7. Configuration (`internal/settings/`)

| Setting key | Type | Default | Description |
|-------------|------|---------|-------------|

## 8. Wiring and dependencies

- `module.go`: fx providers, DB/Redis init, stub providers for other services
- External features consumed (stub paths)
- Characteristics applied (cache, workflow, i18n, telemetry)

## 9. Testing design

| Layer        | Approach                | Key scenarios |
|--------------|-------------------------|---------------|
| Domain / app | unit tests              |               |
| API / repo   | `x/testkit` integration |               |

## 10. Open issues and revision history

| Version | Date | Author | Changes |
|---------|------|--------|---------|
