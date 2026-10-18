---
name: check_frontend_design
---

Verify that a frontend design document (`docs/design/<app>-frontend-design.md`) exists, has frontmatter `status: approved`, contains all required design-system sections, and has no leftover instructional comments before initializing a frontend stack (`create-next-app`, `create vite`) or writing UI code.

## When to use

- After the user approves `docs/design/<app>-frontend-design.md`, before running `create-next-app`, `create vite`, or writing UI components.
- During `review-frontend` audits to confirm the app's design document is complete and approved.

## Extended Guidance

| Argument        | Default      | Description                                                                                      |
|-----------------|--------------|--------------------------------------------------------------------------------------------------|
| `workspacePath` | (required)   | Path to the workspace root (or `frontend/` / `backend/` subdirectory in a fullstack workspace).  |
| `app`           | `"web"`      | Frontend app slug (`"web"` for a single app at `frontend/`, or `<app>` for `frontend/<app>/`).   |

### What the gate checks

1. **File existence:** `docs/design/<app>-frontend-design.md` exists.
2. **Approval status:** YAML frontmatter declares `status: approved` (only the user sets `status: approved`; agents write `status: draft`).
3. **Required sections:** Markdown headings exist for `Context`, `Aesthetic direction`, `Token plan`, `Layout`, `Component rules`, `Motion`, `v1 surfaces`, and `Quality floor` (matching `knowledge://ronyup/architecture/frontend-design-template`).
4. **No placeholder comments:** No leftover `<!-- ... -->` template comments remain in the document.
