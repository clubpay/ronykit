# ronykit-framework Skill

[Agent Skills](https://agentskills.io/specification)-style skill that **orchestrates**
RonyKit development. Domain knowledge lives in the **`ronyup` MCP server** (`ronyup mcp`).

## Contents

| Path | Role |
|------|------|
| `SKILL.md` | Playbook: prerequisites, workflows, hard rules, validation |
| `references/mcp-map.md` | Index of MCP tools, prompts, and knowledge resources |

## Locations (project)

Installed in both standard skill roots (same content):

- `.cursor/skills/ronykit-framework/` — Cursor
- `.agents/skills/ronykit-framework/` — Claude Code, Codex, and other Agent Skills clients

Invoke in Cursor: `/ronykit-framework`

## MCP

```json
{
  "mcpServers": {
    "ronyup": {
      "command": "ronyup",
      "args": ["mcp"]
    }
  }
}
```

(Cursor: `.cursor/mcp.json`; scaffolded workspaces also ship `.ai/mcp/mcp.json`.)

## Install globally

```bash
SKILL_ROOT=ronykit-framework
for base in ~/.cursor/skills ~/.agents/skills; do
  mkdir -p "$base/$SKILL_ROOT/references"
  cp .cursor/skills/ronykit-framework/SKILL.md "$base/$SKILL_ROOT/"
  cp .cursor/skills/ronykit-framework/references/mcp-map.md "$base/$SKILL_ROOT/references/"
done
```

## Scaffolded workspaces

`ronyup setup workspace` copies `.cursor/skills/`, `.agents/skills/`, and MCP config
into the new repository.

## Maintenance

- **Conventions / architecture:** `ronyup/cmd/mcp/knowledge/` only.
- **Agent workflows / MCP index:** this skill (`SKILL.md`, `references/mcp-map.md`).
