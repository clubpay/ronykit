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

- An approved SRS must exist at `docs/design/{{feature_name}}-srs.md` {{#if srs_path}}(or at {{srs_path}}){{/if}}.
- Read the SRS fully before designing. Trace every design element back to SRS requirement IDs.

## Instructions

1. Read the SRS file in the workspace.
2. Read `knowledge://ronyup/architecture/design-documents`, `sdd-template`, `service-structure`, `api-handler-files`, `domain-layer`, `repo-ports`, `postgres-sqlc`, `module-wiring`, `settings-config`, and `error-handling`.
3. For each characteristic in the SRS or arguments, read `knowledge://ronyup/characteristics/<name>`.
4. Write the SDD to **`docs/design/{{feature_name}}-sdd.md`**.

## SDD content rules

- Follow IEEE 1016–2009 structure mapped to RonyKIT modules (see `sdd-template` resource).
- Include a **requirement traceability** table (SRS FR/NFR → design element → file path).
- Specify every API operation: route, handler name, input/output fields, domain errors.
- Define domain types and `internal/domain/errors.go` codes (`SCREAMING_SNAKE_CASE`).
- Define repository port interfaces and schema/migration outline.
- List settings keys and fx wiring notes (`module.go`, stub providers).
- Describe testing approach (unit vs `x/testkit` integration).

## Output

- Create or update `docs/design/{{feature_name}}-sdd.md`.
- Summarize key design decisions and any SRS gaps found.
- **Stop here.** Do not scaffold or implement until the user reviews and approves the SDD.
- When approved, run `scaffold_feature`, then `write-service-code` using the SDD as the source of truth.
