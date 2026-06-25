# Design

This directory holds design documents for features and frontends.

| Document | File pattern |
|----------|--------------|
| SRS | `<feature>-srs.md` |
| SDD | `<feature>-sdd.md` |
| Frontend design | `<app>-frontend-design.md` |

**Standards:** IEEE 830 (SRS), IEEE 1016–2009 (SDD).

**Backend agent workflow:** MCP prompts `write-srs` → `write-sdd` → `scaffold_feature` → `write-service-code` — or orchestration prompt
`design-new-service`. Read `knowledge://ronyup/architecture/design-documents` and `srs-template` / `sdd-template` before writing.

**Frontend agent workflow:** MCP prompt `design-frontend` (ask design questions → write doc → user approval → init stack → implement). Read
`knowledge://ronyup/architecture/frontend-design-documents` and `frontend-design-template`. Skills: `frontend-design`, `design-tokens`,
`typography`.

**Approval gates (enforced):**

- Backend: `scaffold_feature` refuses until `<feature>-srs.md` and `<feature>-sdd.md` have `status: approved`.
- Frontend: `frontend/verify.sh` refuses until `<app>-frontend-design.md` has `status: approved` (once the app is initialized).
- Backend tests: `verify.sh` (and Cursor `backend-verify` hook) require integration tests for every repo port method and unit tests for every
  `App` method.

Each document starts with YAML frontmatter `status: draft`. Only the user changes it to `status: approved`.
