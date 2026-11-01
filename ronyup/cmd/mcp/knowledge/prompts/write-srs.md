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
3. For each requested characteristic, read `knowledge://ronyup/characteristics/<name>` (URI = filename: `cache`, `redis`, `database`, `api`,
   `testing`, not the first YAML keyword).
4. Read relevant `knowledge://ronyup/architecture/*` resources (e.g. `inter-service-stubs`, `postgres-sqlc`, `flow-workflows`, `table-partitioning`) when the requirements imply them — especially `table-partitioning` when data volume or retention will grow over time.
5. Ask clarifying questions if requirements are ambiguous — do not invent unstated behavior. If the direction needs stress-testing, use the optional six-lens brainstorm in `.agents/skills/ronykit-framework/references/egm.md` only after the user confirms the anchored hypothesis.
6. Write the SRS to **`docs/design/{{feature_name}}-srs.md`** in the workspace root.
7. Read `knowledge://ronyup/architecture/egm-design-review` and run fresh-context Goldfish zoom-out and comprehension reviews. Write reports to `docs/design/reviews/{{feature_name}}-srs/`, classify findings with the user, and apply only approved document edits.

## SRS content rules

- Begin the file with the required YAML frontmatter (see `srs-template`). Set `status: draft`. **Never** set `status: approved` yourself — only the user approves it. The `scaffold_feature` tool is gated on this status, so an unapproved SRS blocks scaffolding and implementation.
- Follow IEEE 830 structure (see template resource).
- Number functional requirements (FR-001, …) and non-functional requirements (NFR-001, …).
- Every FR must have acceptance criteria.
- Include an **API requirements** table: operation name, HTTP method + path (or RPC), auth, summary.
- State persistence needs, inter-service dependencies, and configuration expectations. For append-only or high-volume tables, capture retention, expected growth, and whether partitioning is required (NFR) so SDD can choose monthly/quarterly time partitions and automated maintenance.
- List open questions in an appendix when information is missing.

## Output

- Create or update `docs/design/{{feature_name}}-srs.md` with `status: draft` frontmatter and write the EGM review artifacts.
- Summarize what you wrote, reviewer status, unresolved findings, and open questions. If fresh subagents are unavailable, mark EGM `not run` and ask whether the user wants human-only review; never substitute a same-thread review.
- **Stop here.** Do not write the SDD, scaffold, or code until the user reviews and approves the SRS.
- Approval = the user changes the frontmatter to `status: approved` (or explicitly tells you to). Do not approve it on your own.
- When approved, continue with the `write-sdd` prompt (or `design-new-service` for the full workflow).
