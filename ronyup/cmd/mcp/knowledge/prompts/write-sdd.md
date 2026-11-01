---
name: write-sdd
description: Write a Software Design Description (SDD) for a RonyKIT service feature based on an approved SRS.
arguments:
- name: feature_name
  description: Feature directory name (e.g. billing, auth).
  required: true
- name: srs_path
  description: "Path to the approved SRS file (default docs/design/<feature_name>-srs.md)."
  required: false
- name: characteristics
  description: "Comma-separated traits (e.g. postgres, cache, workflow, i18n)."
  required: false
---

You are writing the **Software Design Description (SDD)** for the RonyKIT service feature **{{feature_name}}mod**.

{{#if srs_path}} SRS file: {{srs_path}} {{/if}} {{#if characteristics}} Requested characteristics: {{characteristics}} {{/if}}

## Prerequisites

- An **approved** SRS must exist at `docs/design/{{feature_name}}-srs.md` {{#if srs_path}}(or at {{srs_path}}){{/if}}. Its frontmatter must contain `status: approved`. If it is still `draft`, stop and ask the user to approve the SRS.
- Read the full SRS before designing. Trace every design element back to SRS requirement IDs.

## Instructions

1. Read the SRS file in the workspace.
2. Read `knowledge://ronyup/architecture/design-documents`, `sdd-template`, `service-structure`, `api-handler-files`, `domain-layer`, `repo-ports`, `postgres-sqlc`, `module-wiring`, `settings-config`, and `error-handling`.
3. If the SRS implies continuous data growth, also read `knowledge://ronyup/architecture/table-partitioning`. Examples include events, audit records, ledgers, logs, and high write volume. Complete SDD section 6.2 with strategy, granularity, ahead window, retention, and maintenance scheduler.
4. For each characteristic in the SRS or arguments, read `knowledge://ronyup/characteristics/<name>`. The URI uses the filename, such as `cache`, `redis`, `database`, or `api`. Do not use the first keyword.
5. Read `knowledge://ronyup/architecture/package-selection`. List the RonyKIT package imports in the SDD wiring section. These imports can include `rony`, `rony/errs`, `x/*`, and `flow`. Open each matching `knowledge://ronyup/packages/<name>` before specifying constructors.
6. Write the SDD to **`docs/design/{{feature_name}}-sdd.md`**.
7. Read `knowledge://ronyup/architecture/egm-design-review`. Run all five EGM stages with fresh read-only Goldfish subagents: initial zoom out, comprehension, design critique, intermediate zoom out, and implementation readiness.
8. Write reports to `docs/design/reviews/{{feature_name}}-sdd/`. Classify findings with the user. Apply only approved document edits.

## SDD content rules

- Begin with the required YAML frontmatter from `sdd-template`. Set `status: draft`. Only the user sets `status: approved`. `scaffold_feature` enforces this status. An unapproved SDD blocks scaffolding and implementation.
- Follow the IEEE 1016–2009 structure mapped to RonyKIT modules in `sdd-template`.
- Include a **requirement traceability** table: SRS FR/NFR, design element, and file path.
- Record viable rejected alternatives with concrete reasons. Include applicable failure modes, security and scalability risks, constraints, and mitigations.
- Specify every API operation: route, handler name, input and output fields, and domain errors.
- Define domain types and `internal/domain/errors.go` codes in `SCREAMING_SNAKE_CASE`.
- Define repository port interfaces and the schema and migration outline.
- List settings keys and fx wiring notes for `module.go` and stub providers.
- Describe the testing approach for unit and `x/testkit` integration tests.

## Output

- Create or update `docs/design/{{feature_name}}-sdd.md` with `status: draft`. Write the EGM review artifacts.
- Summarize key design decisions, reviewer status, unresolved findings, and SRS gaps. If fresh subagents are unavailable, mark EGM `not run`. Ask whether the user wants human-only review. Never substitute a same-thread review.
- **Stop here.** Do not scaffold or implement until the user reviews and approves the SDD.
- Approval means the user changes the frontmatter to `status: approved` or explicitly approves the document. Do not approve it yourself.
- `scaffold_feature` requires both SRS and SDD to have `status: approved`. After approval, run `scaffold_feature`, then `write-service-code`. Use the SDD as the source of truth.
