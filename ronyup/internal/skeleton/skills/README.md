# Authoring bundled skills

These are the agent skills `ronyup` installs into a workspace's
`.agents/skills/<id>/`. `.claude/skills/<id>` is a symlink to that directory
so Claude Code can load it. Cursor and Codex read `.agents/skills` directly.
Each directory is a self-contained skill; register it in
the catalog (`ronyup/internal/scaffold/skills.go`) so it can be selected. `copySkills`
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

- Examples: `frontend-design`, `dashboard-ui`, `ui-polish`, `go-modern`, `code-review`.

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
  single giant concatenated document. Architecture and craft skills
  (`release-it`, `domain-driven-design`, `software-design-philosophy`,
  `ddia-systems`, `technical-documentation`) are adapted from
  [`wondelai/skills`](https://github.com/wondelai/skills) (MIT). Frontend UI and
  motion skills (`frontend-design`, `dashboard-ui`, `ux-quality`, `ui-polish`,
  `typography`, `design-tokens`, `nextjs-modern`) synthesize patterns from
  [`jakubkrehel/skills`](https://github.com/jakubkrehel/skills) (MIT),
  [`emilkowalski/skills`](https://github.com/emilkowalski/skills) (MIT),
  [`pbakaus/impeccable`](https://github.com/pbakaus/impeccable) (Apache-2.0),
  [`nexu-io/open-design`](https://github.com/nexu-io/open-design) (MIT),
  [`Dammyjay93/interface-design`](https://github.com/Dammyjay93/interface-design) (MIT),
  [`ibelick/ui-skills`](https://github.com/ibelick/ui-skills), and
  [`vercel-labs/web-interface-guidelines`](https://github.com/vercel-labs/web-interface-guidelines).
  The `ronykit-framework` EGM protocol and reviewer rubrics are adapted from
  [`google/elephant-goldfish-model`](https://github.com/google/elephant-goldfish-model)
  (Apache-2.0); preserve its pinned commit attribution and bundled license.
- Only vendor a skill whose core ideas are language-neutral. Class-bound
  material (inheritance refactorings, SOLID-per-class, Java seams) does not
  translate to Go; distill what survives into a Go-native skill instead of
  converting it line by line. That is why `go-design` replaces the upstream
  `clean-architecture`, `refactoring-patterns`, and `working-with-legacy-code`.
  They intentionally diverge from upstream: the book concepts stay, but every
  code example (in `SKILL.md` and `references/`) is idiomatic Go shaped like a
  RonyKit feature module (`internal/domain`, `internal/app`, `repo/port.go`,
  `rony/errs`, `fx`) or TypeScript for Next.js/React, and advice that
  contradicts RonyKit conventions has been rewritten. Do not re-sync by
  copying upstream files over ours. To pick up upstream changes, diff the new
  upstream against the previous upstream version, then port only the conceptual
  changes, keeping the Go/TS examples. Check Go snippets parse and are
  gofmt-clean before committing.
- Descriptions route to siblings ("For X, see Y") so overlapping skills
  (`writing-tests`/`go-testing`, `code-review`/`go-design`) don't
  compete for the same trigger.
- Only name sibling skills that are bundled here; an unbundled name is a dead
  pointer for the agent.
- Wrap prose at ~80 columns to match the other skills.

## Upstream baselines

Record the exact source revision used for every synchronization. These are review
baselines, not instructions to overwrite the RonyKit adaptations. On the next
audit, diff the new upstream against these revisions, port conceptual changes,
and update the entries in the same change.

| Source | Revision | Bundled skills |
| ------ | -------- | -------------- |
| `wondelai/skills` | `c172996495bed0fcd26896a9416b2093fd7073f0` | `go-design`, `release-it`, `domain-driven-design`, `software-design-philosophy`, `ddia-systems`, `technical-documentation` |
| `obra/superpowers` | `2ce8aa15d05bd57213e6f746f953aafb7ded7a8b` | `verification-before-completion`, `writing-plans` |
| `DaleStudy/skills` | `10d2ee2e27c71052e64e29a640f2d178ff11b253` | `storybook` |
| `vercel-labs/agent-skills` | `1a2c79d4e6c4a2283dc2c614b6538cbe1ed5ca62` | `react-performance`, `composition-patterns` |
| `anthropics/skills` (`skills/frontend-design`) | `ca45fc496a3436251f191354091d557da31c15b5` | `frontend-design` |
| `pbakaus/impeccable` | `e89e3ada3e875a89ac2c5d33d11e65ebef6fe285` | `frontend-design`, `dashboard-ui`, `ui-polish`, `typography` |
| `nexu-io/open-design` (`plugins/community/hallmark`) | `12d970eacc8971150e3c76470d2a9afe16dc2ed1` | `frontend-design` |
| `Dammyjay93/interface-design` | `2f9be3206855bcb2d1d0af262c8bae25cba6658d` | `dashboard-ui`, `design-tokens` |
| `vercel-labs/web-interface-guidelines` | `434b7f91364665f2f733b310ec54809bf8f37937` | `ux-quality` |
| `ibelick/ui-skills` | `4907a526b07943b3a338177558d42fae4fdf0a86` | `ux-quality`, `ui-polish` |
| `jakubkrehel/skills` | `592aedc44d7d5b3e29740984252902f3e5402a1c` | `ux-quality`, `ui-polish`, `typography`, `design-tokens` |
| `emilkowalski/skills` (`improve-animations`) | `a81dbca07f2217b7103e510c0f13f8d5b017afc9` | `ui-polish` |
| `google/elephant-goldfish-model` | `403d8881d62022490637d740657b33f01e321ea8` | `ronykit-framework` EGM assets |
| `WebMCP-org/npm-packages` | `469bad289e3aee1d19a487c2c7820d2c54a82373` | `webmcp` |
| `nextlevelbuilder/ui-ux-pro-max-skill` | `58ff1f32156cbe144108de48eda2ecaed9a` | `shadcn` references |

Live APIs do not have a repository baseline in every skill. The current review
uses Next.js **16.4.3**, React **19.3.1**, Storybook **10.6.5**, shadcn
**4.23.0**, MCP-B packages **6.0.3**, Zod **4.6.7** (with a **4.2+** floor),
the W3C WebMCP draft, and the Chrome WebMCP guide. Recheck moving documentation
when updating `nextjs-modern`, `storybook`, `shadcn`, or `webmcp`.

## Registering a skill

Add an entry to `skillCatalog` in `ronyup/internal/scaffold/skills.go` (id, name,
one-line description, `Category`, and the `DefaultBackend`/`DefaultFullstack`
flags). `TestCatalogSkillsExistInEmbedFS` verifies every catalog entry has an
embedded `SKILL.md`.
