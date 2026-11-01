---
name: sync_skills
---

Synchronize agent skills from the catalog embedded in the running `ronyup` MCP server.

## When to use

- `validate_skills` reports missing or outdated skill files.
- The workspace was created by an older `ronyup` and its installed skills need updates.
- Claude skill links, retired skills, or legacy Cursor copies need migration.

This tool only changes `.agents/skills/`, `.claude/skills/`, and legacy `.cursor/skills/` entries. Use `sync_workspace` for AGENTS.md, hooks, Makefiles, devops assets, and other boilerplate.

## Arguments

| Argument | Default | Behavior |
|----------|---------|----------|
| `workspacePath` | required | Repository root, or `backend/` in a fullstack workspace |
| `kind` | `auto` | Workspace layout override |
| `skills` | `installed` | `installed`, `default`, `all`, `none`, or explicit skill IDs |
| `overwrite` | `false` | Replace modified files, repair noncanonical links, and remove retired or legacy copies |

The `ronykit-framework` skill is always included. With `overwrite: false`, sync only adds missing files and links; it never replaces or deletes existing content. Extra user-authored files are preserved in either mode.

The tool returns a structured post-sync `validation` report. If drift remains and overwrite was false, review the reported paths and rerun with `overwrite: true` when replacement is intended.

CLI equivalent for the sync operation:

```bash
ronyup setup sync --only skills --skills installed [--overwrite]
```
