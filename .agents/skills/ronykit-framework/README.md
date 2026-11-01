# ronykit-framework Skill

[Agent Skills](https://agentskills.io/specification)-style skill that **orchestrates** RonyKit development. Domain knowledge lives in the **`ronyup` MCP server** (`ronyup mcp`).

## Contents

| Path                         | Role                                                            |
|------------------------------|-----------------------------------------------------------------|
| `SKILL.md`                   | Playbook: prerequisites, workflows, hard rules, validation      |
| `references/mcp-map.md`      | Index of MCP tools, prompts, and knowledge resources            |
| `references/egm.md`          | Elephant–Goldfish design, review, resolution, and drift protocol |
| `references/egm-reviewers.md` | Goldfish reviewer lenses and rubrics                            |
| `references/egm-LICENSE`     | Apache-2.0 license for the adapted EGM material                 |

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
mkdir -p ~/.agents/skills/$SKILL_ROOT
cp .agents/skills/ronykit-framework/SKILL.md ~/.agents/skills/$SKILL_ROOT/
cp -R .agents/skills/ronykit-framework/references ~/.agents/skills/$SKILL_ROOT/
```

Copy the complete `references/` directory: the EGM workflow needs its reviewer
rubrics and license as well as the MCP index.

## Scaffolded workspaces

`ronyup setup workspace` installs the selected skills under `.agents/skills/`. Cursor and Codex read that tree. `.claude/skills/<id>` is a symlink to the same directory so Claude Code can discover it. MCP config is copied into the new repository.

## Maintenance

- **Conventions / architecture:** `ronyup/cmd/mcp/knowledge/` only.
- **Agent workflows / MCP index / EGM integration:** this skill (`SKILL.md`,
  `references/`). Keep the monorepo and shipped skeleton copies synchronized.
- EGM material is adapted from
  [`google/elephant-goldfish-model`](https://github.com/google/elephant-goldfish-model)
  (Apache-2.0). Preserve the pinned source commit and `references/egm-LICENSE`.
