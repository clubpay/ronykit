# Design

This directory holds Software Requirements Specification (SRS) and Software Design Description (SDD) documents for features being developed.

| Document | File pattern       |
|----------|--------------------|
| SRS      | `<feature>-srs.md` |
| SDD      | `<feature>-sdd.md` |

**Standards:** IEEE 830 (SRS), IEEE 1016–2009 (SDD).

**Agent workflow:** Use MCP prompts in order — `write-srs` → `write-sdd` → `scaffold_feature` → `write-service-code` — or the orchestration prompt `design-new-service`. Read `knowledge://ronyup/architecture/design-documents` and the `srs-template` / `sdd-template` resources before writing documents.

**Approval gate (enforced):** Each document must start with YAML frontmatter `status: draft`. Only the user changes it to `status: approved`. The `scaffold_feature` tool refuses to run until both `<feature>-srs.md` and `<feature>-sdd.md` are `status: approved`, so you cannot scaffold or implement until SRS and SDD are approved.
