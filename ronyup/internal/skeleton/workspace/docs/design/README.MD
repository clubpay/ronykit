# Design

This directory holds Software Requirements Specification (SRS) and Software Design Description (SDD) documents for features being developed.

| Document | File pattern       |
|----------|--------------------|
| SRS      | `<feature>-srs.md` |
| SDD      | `<feature>-sdd.md` |

**Standards:** IEEE 830 (SRS), IEEE 1016–2009 (SDD).

**Agent workflow:** Use MCP prompts in order — `write-srs` → `write-sdd` → `scaffold_feature` → `write-service-code` — or the orchestration prompt `design-new-service`. Read `knowledge://ronyup/architecture/design-documents` and the `srs-template` / `sdd-template` resources before writing documents. Do not scaffold or implement until SRS and SDD are approved.
