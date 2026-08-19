@AGENTS.md

<!--
Claude Code reads CLAUDE.md; Cursor and other assistants read AGENTS.md. To keep both
tools on the same guidance, the shared instructions live in AGENTS.md and are imported
above — do not duplicate them here. Add content below only if it is specific to the
Claude Code mechanism itself, never repo rules (those belong in AGENTS.md so every tool
gets them).

Shared skills ship under .agents/skills/ and are routed from AGENTS.md by path — read
them there. Claude-Code-specific config, when you need it, lives under:
  .claude/rules/         path-scoped rules (frontmatter `paths:`)
  .claude/settings.json  permissions / deny-read rules
-->
