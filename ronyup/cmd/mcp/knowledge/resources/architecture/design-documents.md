# Design documents (SRS and SDD)

Before scaffolding or writing service code, capture requirements and design in `docs/design/` at the workspace root. Use MCP prompts in order:

1. `write-srs` — Software Requirements Specification (IEEE 830 style), followed by EGM requirements review and user approval
2. `write-sdd` — Software Design Description (IEEE 1016–2009 style), derived from the approved SRS, followed by full EGM review and user approval
3. `scaffold_feature` — create the module skeleton
4. `write-service-code` — implement code that matches the SDD

## File naming

| Document | Path                                |
|----------|-------------------------------------|
| SRS      | `docs/design/<feature_name>-srs.md` |
| SDD      | `docs/design/<feature_name>-sdd.md` |

Use the feature directory name (e.g. `billing`, `auth`), not the Go package suffix (`billingmod`).

## Approval status (frontmatter)

Both documents **must** begin with YAML frontmatter that carries an approval status:

```yaml
---
feature: <feature_name>
document: SRS   # or SDD
status: draft   # set to "approved" only after the user reviews
---
```

Agents write documents with `status: draft`. **Only the user** flips a document to `status: approved` (by editing the file or explicitly telling the agent to). Agents must never approve documents on their own behalf.

## Gate rules (enforced by the tool)

- Do **not** write the SDD until the SRS is complete and the user has approved it (`status: approved`).
- Do **not** scaffold or implement code until the SDD is complete and approved.
- The `scaffold_feature` tool **enforces** this: it returns an error and refuses to run unless both `docs/design/<feature>-srs.md` and `docs/design/<feature>-sdd.md` exist with `status: approved`. This is a hard gate, not a convention.
- The only bypass is `skipDesignGate=true` on `scaffold_feature`, which must be used **only** when the user explicitly asks to skip the design documents.
- When implementation diverges from the SDD, update the SDD first (or note the change in the SDD revision history).

## EGM review before approval

Read `knowledge://ronyup/architecture/egm-design-review` and
`.agents/skills/ronykit-framework/references/egm.md` before requesting approval.
The Elephant drafts in the primary context; independent Goldfish subagents
review with no drafting conversation; the user resolves findings and approves.

- **SRS:** run zoom-out and comprehension review. Do not introduce design
  alternatives during requirements review.
- **SDD:** run all five stages: initial zoom out, comprehension, design
  critique, intermediate zoom out, and implementation readiness.
- Write findings under `docs/design/reviews/<document-slug>/`, classify them
  with the user, and apply only approved document edits.
- If fresh subagents are unavailable, mark the review `not run` and ask whether
  to continue with human-only review. Never label a same-thread review EGM.

EGM review is part of the agent workflow, not a replacement for the existing
`status: approved` enforcement or test gates.

## SRS must cover

- Purpose, scope, definitions, references
- Overall description (product perspective, functions, user classes, constraints, assumptions)
- Specific requirements: functional, non-functional (performance, security, reliability), external interfaces
- For RonyKIT services: list API operations (names + brief behavior), persistence needs, inter-service dependencies, and requested characteristics (postgres, cache, workflow, i18n, etc.)

## SDD must cover

- Design overview and context (reference the SRS file path)
- Module decomposition mapped to RonyKIT layout (`api/`, `internal/app/`, `internal/domain/`, `internal/repo/`, `internal/settings/`)
- Alternatives considered and concrete rejection reasons
- Failure modes, security and scalability risks, constraints, and mitigations
- API contracts: routes, HTTP methods, input/output field lists, error codes
- Domain model: entities, value objects, domain errors
- Repository ports and persistence (tables, queries, migrations outline)
- Settings and configuration keys
- Cross-service stub usage (which other features are called)
- Testing strategy (unit vs integration)

## Template resources

Read these before writing documents:

- `knowledge://ronyup/architecture/srs-template` — SRS section outline
- `knowledge://ronyup/architecture/sdd-template` — SDD section outline mapped to RonyKIT modules
- `knowledge://ronyup/architecture/egm-design-review` — independent design-review workflow before approval

## Orchestration prompt

For end-to-end work, start with the `design-new-service` MCP prompt. It runs the full SRS → SDD → scaffold → implement workflow with explicit phase gates.
