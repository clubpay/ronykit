# Authoring bundled skills

These are the agent skills `ronyup` installs into a workspace's
`.agents/skills/<id>/`. Each directory is a self-contained skill; register it in
the catalog (`ronyup/cmd/setup/skills.go`) so it can be selected. `copySkills`
copies the **whole** skill directory, so a `rules/` (or `references/`) subfolder
ships automatically — no extra wiring.

This file is a contributor note; it is **not** copied into scaffolded
workspaces.

## How skills load (this drives the structure decision)

Three tiers, by when content enters the model's context:

1. **`description` (frontmatter)** — always in context; the model reads it to
   decide whether to activate the skill. Keep it a sharp, trigger-rich sentence.
2. **`SKILL.md` body** — loaded in full **on activation**. Every token here is
   paid on every use, so keep it focused (aim well under ~500 lines).
3. **Referenced files (`rules/*.md`, `references/*.md`)** — loaded **only when
   the model opens them** (progressive disclosure). Free until needed.

## The structure heuristic — single file vs. rules tree

Decide deliberately; don't copy the shape of a neighboring skill.

**Default to a single self-contained `SKILL.md`** when the material is
judgment/taste guidance that fits comfortably in ~100–200 lines (principles,
tables, a checklist). Splitting these adds indirection with no reliability gain.

- Examples: `frontend-design`, `dashboard-ui`, `go-modern`, `code-review`.

**Use a `rules/` tree (progressive disclosure)** when the material is a large,
example-heavy catalog (many discrete rules, each with concrete code and/or
metrics) that would bloat the body or dilute attention if inlined.

- Example: `react-performance` (70 rules vendored under `rules/`).

When you choose a rules tree, the `SKILL.md` must still **stand on its own for
the common path** — small/weaker models are unreliable at the "notice the
pointer → open the file" step, so:

1. Inline the highest-impact patterns (with minimal code) so a model that never
   opens a rule file still gets the common case right.
2. Give a **deterministic routing table** ("working on X → read `rules/x.md`")
   plus a full one-line index of every rule, so the trigger is explicit.
3. Keep each rule file self-contained (title, impact, incorrect→correct) so a
   partial read is still coherent.

Rule of thumb: would inlining everything push the body well past ~300–400 lines,
or is the value mostly in copy-paste-ready examples? If yes, use a rules tree.
Otherwise, keep it one file.

## Conventions

- Frontmatter: `name` (= directory id) and a folded `description: >-` whose text
  states **what it does** and **when to use it** (trigger phrases).
- Body: a short framing paragraph, a `## When to use` section, focused sections,
  and a `## Checklist before "done"`.
- Cross-reference sibling skills by name instead of duplicating their content.
- Vendored material: keep upstream license/attribution (e.g. a metadata file and
  an `## Attribution` section), and prefer the granular source files over a
  single giant concatenated document. Architecture skills (`clean-architecture`,
  `refactoring-patterns`, `release-it`, `working-with-legacy-code`) are adapted
  from [`wondelai/skills`](https://github.com/wondelai/skills) (MIT).
- Wrap prose at ~80 columns to match the other skills.

## Registering a skill

Add an entry to `skillCatalog` in `ronyup/cmd/setup/skills.go` (id, name,
one-line description, `Category`, and the `DefaultBackend`/`DefaultFullstack`
flags). `TestCatalogSkillsExistInEmbedFS` verifies every catalog entry has an
embedded `SKILL.md`.
