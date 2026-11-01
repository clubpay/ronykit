# Design

This directory contains design documents for features and frontends.

| Document | File pattern |
|----------|--------------|
| Optional EGM concept brief | `<feature>-concept.md` |
| SRS | `<feature>-srs.md` |
| SDD | `<feature>-sdd.md` |
| Frontend design | `<app>-frontend-design.md` |
| EGM review reports | `reviews/<document-slug>/` |

**Standards:** IEEE 830 for SRS and IEEE 1016–2009 for SDD. The Elephant–Goldfish Model (EGM) provides independent design review.

## Backend agent workflow

1. Read `knowledge://ronyup/architecture/design-documents`, `egm-design-review`, `srs-template`, and `sdd-template`.
2. Use MCP prompt `write-srs`.
3. Run the EGM requirements review and get user approval.
4. Use MCP prompt `write-sdd`.
5. Run the full EGM review and get user approval.
6. Call `scaffold_feature`, then use `write-service-code`.

MCP prompt `design-new-service` coordinates the same workflow.

## Frontend agent workflow

1. Read `knowledge://ronyup/architecture/frontend-design-documents`, `egm-design-review`, and `frontend-design-template`. Read skills `frontend-design`, `design-tokens`, and `typography`.
2. Use MCP prompt `design-frontend` to ask design questions and write the document.
3. Run the EGM review and get user approval.
4. Initialize the frontend stack and implement the design.

## EGM review rules

The Elephant drafts in the primary agent context. Each Goldfish reviewer is a fresh read-only subagent without the drafting conversation.

A same-thread review is not EGM. If delegation is unavailable, mark the review `not run`. Ask whether to continue with human-only review.

See `.agents/skills/ronykit-framework/references/egm.md` for the complete protocol.

## Approval gates

- **Backend:** `scaffold_feature` requires `status: approved` in `<feature>-srs.md` and `<feature>-sdd.md`.
- **Frontend:** After app initialization, `frontend/verify.sh` requires `status: approved` in `<app>-frontend-design.md`.
- **Backend tests:** `verify.sh` and the Cursor `backend-verify` hook require an integration test for every repository port method. They also require a unit test for every `App` method.

Each document starts with YAML frontmatter `status: draft`. Only the user changes it to `status: approved`.
