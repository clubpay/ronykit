# Design documents (SRS and SDD)

Before scaffolding or writing service code, capture requirements and design in `docs/design/` at the workspace root. Use MCP prompts in order:

1. `write-srs` — Software Requirements Specification (IEEE 830 style)
2. `write-sdd` — Software Design Description (IEEE 1016–2009 style), derived from the SRS
3. `scaffold_feature` — create the module skeleton
4. `write-service-code` — implement code that matches the SDD

## File naming

| Document | Path                                |
|----------|-------------------------------------|
| SRS      | `docs/design/<feature_name>-srs.md` |
| SDD      | `docs/design/<feature_name>-sdd.md` |

Use the feature directory name (e.g. `billing`, `auth`), not the Go package suffix (`billingmod`).

## Gate rules

- Do **not** write the SDD until the SRS is complete and the user has reviewed it.
- Do **not** scaffold or implement code until the SDD is complete and the user has reviewed it.
- When implementation diverges from the SDD, update the SDD first (or note the change in the SDD revision history).

## SRS must cover

- Purpose, scope, definitions, references
- Overall description (product perspective, functions, user classes, constraints, assumptions)
- Specific requirements: functional, non-functional (performance, security, reliability), external interfaces
- For RonyKIT services: list API operations (names + brief behavior), persistence needs, inter-service dependencies, and requested characteristics (postgres, cache, workflow, i18n, etc.)

## SDD must cover

- Design overview and context (reference the SRS file path)
- Module decomposition mapped to RonyKIT layout (`api/`, `internal/app/`, `internal/domain/`, `internal/repo/`, `internal/settings/`)
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

## Orchestration prompt

For end-to-end work, start with the `design-new-service` MCP prompt. It runs the full SRS → SDD → scaffold → implement workflow with explicit phase gates.
