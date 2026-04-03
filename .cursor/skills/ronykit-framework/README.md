# ronykit-framework Skill

Reusable Cursor skill for building and modifying services in the RonyKit ecosystem.

## What this skill covers

- `rony/` high-level service development
- `kit/` low-level primitives and customization
- `ronyup` scaffolding and MCP workflows
- `std/gateways`, `std/clusters`, `stub`, and `flow` module context
- RonyKit-specific guardrails and validation commands

## Local usage

This repository already includes the skill at:

`./.cursor/skills/ronykit-framework/SKILL.md`

In Cursor chat, invoke it with:

`/ronykit-framework`

## Install globally for your machine

Copy the skill to your personal Cursor skills directory:

```bash
mkdir -p ~/.cursor/skills/ronykit-framework
cp .cursor/skills/ronykit-framework/SKILL.md ~/.cursor/skills/ronykit-framework/SKILL.md
```

Then the skill is available across projects on your machine.

## Share publicly

1. Push this folder to a public Git repository.
2. Share installation steps that copy `SKILL.md` into `~/.cursor/skills/ronykit-framework/`.
3. Optionally publish as part of a Cursor plugin package for marketplace/community distribution.

Reference docs:

- https://cursor.com/docs/context/skills
- https://cursor.com/docs/plugins

## Versioning recommendation

Use semantic tags for this skill (for example `v1.0.0`) and keep a short changelog in commit messages or release notes.
