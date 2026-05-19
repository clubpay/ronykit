# ronykit-framework Skill

Cursor skill that **orchestrates** RonyKit development. It does not duplicate the
knowledge base — that lives in the **`ronyup` MCP server** (`ronyup mcp`).

## Contents

| File | Role |
|------|------|
| `SKILL.md` | Playbook: prerequisites, workflows, hard rules, validation |
| `mcp-map.md` | Index of MCP tools, prompts, and knowledge resources |

## Local usage (this repo)

```text
.cursor/skills/ronykit-framework/
```

In Cursor chat: `/ronykit-framework`

Enable MCP (repo or project):

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

## Install globally

```bash
mkdir -p ~/.cursor/skills/ronykit-framework
cp .cursor/skills/ronykit-framework/SKILL.md ~/.cursor/skills/ronykit-framework/
cp .cursor/skills/ronykit-framework/mcp-map.md ~/.cursor/skills/ronykit-framework/
```

## Scaffolded workspaces

New workspaces created with `ronyup setup workspace` include this skill under
`.cursor/skills/ronykit-framework/` and MCP config under `.ai/mcp/` and `.cursor/`.

## Maintenance

- **Conventions / architecture text:** edit `ronyup/cmd/mcp/knowledge/` only.
- **Agent workflows / MCP index:** edit this skill when tools, prompts, or recommended
  read order change.
