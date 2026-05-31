---
name: write-srs
description: Write a Software Requirements Specification (SRS) for a new RonyKIT service feature before design or implementation.
arguments:
- name: feature_name
  description: Feature directory name (e.g. billing, auth) — not the Go package suffix.
  required: true
- name: requirements
  description: User-provided requirements, goals, constraints, and context for the feature.
  required: true
- name: characteristics
  description: "Comma-separated traits (e.g. postgres, cache, workflow, i18n, idempotent, rest-api)."
  required: false
---

You are writing the **Software Requirements Specification (SRS)** for a new RonyKIT service feature named **{{feature_name}}**.

User requirements and context:

{{requirements}}

{{#if characteristics}} Requested characteristics: {{characteristics}} {{/if}}

## Instructions

1. Read `knowledge://ronyup/architecture/design-documents` for document locations and gate rules.
2. Read `knowledge://ronyup/architecture/srs-template` for the section outline.
3. For each requested characteristic, read `knowledge://ronyup/characteristics/<name>`.
4. Read relevant `knowledge://ronyup/architecture/*` resources (e.g. `inter-service-stubs`, `postgres-sqlc`, `flow-workflows`) when the requirements imply them.
5. Ask clarifying questions if requirements are ambiguous — do not invent unstated behavior.
6. Write the SRS to **`docs/design/{{feature_name}}-srs.md`** in the workspace root.

## SRS content rules

- Follow IEEE 830 structure (see template resource).
- Number functional requirements (FR-001, …) and non-functional requirements (NFR-001, …).
- Every FR must have acceptance criteria.
- Include an **API requirements** table: operation name, HTTP method + path (or RPC), auth, summary.
- State persistence needs, inter-service dependencies, and configuration expectations.
- List open questions in an appendix when information is missing.

## Output

- Create or update `docs/design/{{feature_name}}-srs.md`.
- Summarize what you wrote and list open questions.
- **Stop here.** Do not write the SDD, scaffold, or code until the user reviews and approves the SRS.
- When approved, continue with the `write-sdd` prompt (or `design-new-service` for the full workflow).
