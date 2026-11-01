---
name: design-new-service
description: End-to-end workflow — SRS, then SDD, then scaffold and implement a new RonyKIT service feature.
arguments:
- name: feature_name
  description: Feature directory name (e.g. billing, auth).
  required: true
- name: requirements
  description: User-provided requirements, goals, constraints, and context.
  required: true
- name: characteristics
  description: "Comma-separated traits (e.g. postgres, cache, workflow, i18n, idempotent)."
  required: false
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
2. Ask clarifying questions if needed. If the direction is uncertain, use the optional six-lens EGM brainstorm in `.agents/skills/ronykit-framework/references/egm.md` after explicit user confirmation of the anchored hypothesis.
3. Write the SRS file with the required `status: draft` frontmatter, numbered FR/NFR requirements, and an API requirements table.
4. Read `knowledge://ronyup/architecture/egm-design-review` and run the SRS pipeline: fresh-context Goldfish zoom-out and comprehension reviews. Write reports under `docs/design/reviews/{{feature_name}}-srs/`, then classify and resolve findings with the user.
5. Present a summary and **wait for user approval** before Phase 2.

**Gate:** Do not proceed until the user approves the SRS by setting its frontmatter to `status: approved` (or explicitly telling you to). Never set `status: approved` yourself. If fresh subagents are unavailable, mark EGM `not run` and ask whether to continue with human-only review; never simulate Goldfish in the main context.

---

## Phase 2 — SDD (design)

**Goal:** `docs/design/{{feature_name}}-sdd.md`

1. Read the approved SRS (confirm its frontmatter is `status: approved`).
2. Read `knowledge://ronyup/architecture/sdd-template`, `service-structure`, `api-handler-files`, `domain-layer`, `repo-ports`, `postgres-sqlc`, `module-wiring`, `settings-config`, `error-handling`, and `table-partitioning` when SRS NFRs imply growing data.
3. Write the SDD with the required `status: draft` frontmatter, requirement traceability to the SRS, rejected alternatives with reasons, and risks with mitigations.
4. Run all five EGM stages from `architecture/egm-design-review`: initial zoom out, comprehension, design critique, intermediate zoom out, and implementation readiness. Use fresh read-only subagents, write reports under `docs/design/reviews/{{feature_name}}-sdd/`, and let the user decide which proposed edits to apply.
5. Present a summary and **wait for user approval** before Phase 3.

**Gate:** Do not scaffold or code until the user approves the SDD by setting its frontmatter to `status: approved`. `scaffold_feature` enforces this: it refuses to run unless both the SRS and SDD are `status: approved`. EGM does not replace that gate or any implementation tests.

---

## Phase 3 — Scaffold

1. Call `scaffold_feature` with `featureName={{feature_name}}`, `template=service`, and `characteristics` from the SDD/SRS.
2. Confirm generated paths (`feature/{{feature_name}}/` or grouped layout if specified).

---

## Phase 4 — Implement

1. Follow the SDD as the source of truth.
2. **Before the first `.go` edit**, read `knowledge://ronyup/architecture/package-selection` and every `knowledge://ronyup/packages/<name>`
   you will import (at minimum `rony`, `errs`, `rkit`, plus any `x/` package the SDD uses).
3. Read MCP architecture resources (`service-structure`, `api-handler-files`, `domain-layer`, `repo-ports`, `module-wiring`) before editing
   generated files.
4. Implement in order: `internal/domain/` → `internal/repo/port.go` → `internal/app/` → `api/` → wiring.
5. Write and **run** `x/testkit` integration tests for every repo port method; write unit tests for every `App` method. Read
   `architecture/integration-tests` and skill `go-testing`.
6. Run `make gen-stub` in the feature module after contract changes.
7. Run `make verify` in the Go workspace, then targeted `go test` and workspace `make lint` when feasible.

If implementation reveals design gaps, update the SDD (revision history) before or alongside code changes.

---

## Phase discipline

- Complete one phase at a time unless the user explicitly asks to skip ahead.
- Never skip SRS or SDD for non-trivial features. `scaffold_feature` enforces the gate (approved SRS + SDD); only pass `skipDesignGate=true` when the user explicitly asks to skip the design documents.
- Keep handlers thin; business logic in `internal/app`; persistence behind repo ports.
