# Design

This directory holds design documents for features and frontends.

| Document | File pattern |
|----------|--------------|
| Optional EGM concept brief | `<feature>-concept.md` |
| SRS | `<feature>-srs.md` |
| SDD | `<feature>-sdd.md` |
| Frontend design | `<app>-frontend-design.md` |
| EGM review reports | `reviews/<document-slug>/` |

**Standards:** IEEE 830 (SRS), IEEE 1016–2009 (SDD), with Elephant–Goldfish
Model (EGM) independent design review.

**Backend agent workflow:** MCP prompts `write-srs` → EGM requirements review → user approval → `write-sdd` → full EGM review → user approval →
`scaffold_feature` → `write-service-code` — or orchestration prompt `design-new-service`. Read
`knowledge://ronyup/architecture/design-documents`, `egm-design-review`, and `srs-template` / `sdd-template` before writing.

**Frontend agent workflow:** MCP prompt `design-frontend` (ask design questions → write doc → EGM review → user approval → init stack → implement). Read
`knowledge://ronyup/architecture/frontend-design-documents`, `egm-design-review`, and `frontend-design-template`. Skills: `frontend-design`,
`design-tokens`, `typography`.

The Elephant drafts in the primary agent context. Each Goldfish reviewer is a
fresh read-only subagent with no drafting conversation. Never call a
same-thread review EGM; if delegation is unavailable, mark the review `not run`
and ask whether to continue with human-only review. See
`.agents/skills/ronykit-framework/references/egm.md` for the complete protocol.

**Approval gates (enforced):**

- Backend: `scaffold_feature` refuses until `<feature>-srs.md` and `<feature>-sdd.md` have `status: approved`.
- Frontend: `frontend/verify.sh` refuses until `<app>-frontend-design.md` has `status: approved` (once the app is initialized).
- Backend tests: `verify.sh` (and Cursor `backend-verify` hook) require integration tests for every repo port method and unit tests for every
  `App` method.

Each document starts with YAML frontmatter `status: draft`. Only the user changes it to `status: approved`.
