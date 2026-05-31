---

name: design-new-service description: End-to-end workflow — SRS, then SDD, then scaffold and implement a new RonyKIT service feature. arguments:
- name: feature_name description: Feature directory name (e.g. billing, auth). required: true
- name: requirements description: User-provided requirements, goals, constraints, and context. required: true
- name: characteristics description: "Comma-separated traits (e.g. postgres, cache, workflow, i18n, idempotent)." required: false

---

You are delivering a new RonyKIT service feature **{{feature_name}}** using a phased workflow: **SRS → SDD → scaffold → implement**.

User requirements:

{{requirements}}

{{#if characteristics}} Requested characteristics: {{characteristics}} {{/if}}

Read `knowledge://ronyup/architecture/design-documents` first.

---

## Phase 1 — SRS (requirements)

**Goal:** `docs/design/{{feature_name}}-srs.md`

1. Read `knowledge://ronyup/architecture/srs-template` and relevant characteristics/architecture resources.
2. Ask clarifying questions if needed.
3. Write the SRS file with numbered FR/NFR requirements and an API requirements table.
4. Present a summary and **wait for user approval** before Phase 2.

**Gate:** Do not proceed until the user confirms the SRS is approved.

---

## Phase 2 — SDD (design)

**Goal:** `docs/design/{{feature_name}}-sdd.md`

1. Read the approved SRS.
2. Read `knowledge://ronyup/architecture/sdd-template`, `service-structure`, `api-handler-files`, `domain-layer`, `repo-ports`, `postgres-sqlc`, `module-wiring`, `settings-config`, `error-handling`.
3. Write the SDD with requirement traceability to the SRS.
4. Present a summary and **wait for user approval** before Phase 3.

**Gate:** Do not scaffold or code until the user confirms the SDD is approved.

---

## Phase 3 — Scaffold

1. Call `scaffold_feature` with `featureName={{feature_name}}`, `template=service`, and `characteristics` from the SDD/SRS.
2. Confirm generated paths (`feature/{{feature_name}}/` or grouped layout if specified).

---

## Phase 4 — Implement

1. Follow the SDD as the source of truth.
2. Read MCP architecture resources before editing generated files.
3. Implement in order: `internal/domain/` → `internal/repo/port.go` → `internal/app/` → `api/` → wiring.
4. Run `make gen-stub` in the feature module after contract changes.
5. Add tests per the SDD testing section.
6. Run targeted `go test` and workspace `make lint` when feasible.

If implementation reveals design gaps, update the SDD (revision history) before or alongside code changes.

---

## Phase discipline

- Complete one phase at a time unless the user explicitly asks to skip ahead.
- Never skip SRS or SDD for non-trivial features.
- Keep handlers thin; business logic in `internal/app`; persistence behind repo ports.
