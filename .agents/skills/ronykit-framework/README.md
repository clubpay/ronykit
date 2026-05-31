# ronykit-framework Skill

[Agent Skills](https://agentskills.io/specification)-style skill that **orchestrates** RonyKit development. Domain knowledge lives in the **`ronyup` MCP server** (`ronyup mcp`).

## Contents

| Path                    | Role                                                       |
|-------------------------|------------------------------------------------------------|
| `SKILL.md`              | Playbook: prerequisites, workflows, hard rules, validation |
| `references/mcp-map.md` | Index of MCP tools, prompts, and knowledge resources       |

## Location (project)

Installed under `.agents/skills/ronykit-framework/` ([Agent Skills](https://agentskills.io/specification) layout). Cursor and other compatible agents discover skills from this path automatically.

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
mkdir -p ~/.agents/skills/$SKILL_ROOT/references
cp .agents/skills/ronykit-framework/SKILL.md ~/.agents/skills/$SKILL_ROOT/
cp .agents/skills/ronykit-framework/references/mcp-map.md ~/.agents/skills/$SKILL_ROOT/references/
```

## Scaffolded workspaces

`ronyup setup workspace` copies `.agents/skills/` and MCP config into the new repository.

## Maintenance

- **Conventions / architecture:** `ronyup/cmd/mcp/knowledge/` only.
- **Agent workflows / MCP index:** this skill (`SKILL.md`, `references/mcp-map.md`).
