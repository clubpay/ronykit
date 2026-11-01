---
name: check_frontend_design
---

Verify `docs/design/<app>-frontend-design.md` before initializing a frontend stack or writing UI code. The document must exist and contain `status: approved`. It must include all required design-system sections. No instructional comments can remain.

Framework commands covered by this gate include `create-next-app` and `create vite`.

## When to use

- Use after the user approves `docs/design/<app>-frontend-design.md`. Run before `create-next-app`, `create vite`, or UI components.
- Use during `review-frontend` audits to confirm that the design document is complete and approved.

## Extended Guidance

| Argument | Default | Description |
|----------|---------|-------------|
| `workspacePath` | Required | Path to the workspace root or a `frontend/` or `backend/` subdirectory in a fullstack workspace |
| `app` | `"web"` | Frontend app slug. Use `"web"` for one app at `frontend/` or `<app>` for `frontend/<app>/`. |

### What the gate checks

1. **File existence:** `docs/design/<app>-frontend-design.md` exists.
2. **Approval status:** YAML frontmatter declares `status: approved`. Only the user sets this status. Agents write `status: draft`.
3. **Required sections:** Headings exist for `Context`, `Aesthetic direction`, `Token plan`, `Layout`, `Component rules`, `Motion`, `v1 surfaces`, and `Quality floor`. See `knowledge://ronyup/architecture/frontend-design-template`.
4. **No placeholder comments:** No `<!-- ... -->` template comments remain.
