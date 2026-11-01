---
name: validate_skills
---

Perform a read-only comparison between installed agent skills and the catalog embedded in the running `ronyup` MCP server.

## When to use

Run this before and after skill maintenance, or when an agent needs to know whether its local instructions are current. Use `sync_skills` to repair reported drift.

## Arguments

| Argument | Default | Behavior |
|----------|---------|----------|
| `workspacePath` | required | Repository root, or `backend/` in a fullstack workspace |
| `kind` | `auto` | Workspace layout override |
| `skills` | `installed` | `installed`, `default`, `all`, `none`, or explicit skill IDs |

The `ronykit-framework` skill is always included. The structured result has top-level `upToDate` and per-skill `current` booleans. Each issue includes `kind`, `path`, and `message`.

Validation checks:

- missing skills and required files;
- files whose contents differ from the embedded catalog;
- `SKILL.md` frontmatter name and description;
- unsafe symlink paths and noncanonical Claude links;
- legacy `.cursor/skills/` copies;
- retired skills and their replacements.

Extra user-authored files are allowed and are not deleted by `sync_skills`. Drift is a successful validation result with `upToDate: false`; `isError` is reserved for an invalid workspace or an operational failure.
