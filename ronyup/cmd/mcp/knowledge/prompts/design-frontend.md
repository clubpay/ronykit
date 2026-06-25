---
name: design-frontend
description: >-
  Bootstrap a frontend app with a document-first design workflow — ask aesthetic
  questions, write an approved design doc with tokens and rules, then initialize
  the stack and implement UI.
arguments:
- name: app_name
  description: >-
    Frontend app slug (e.g. web, admin). Use "web" for a single app at frontend/.
  required: true
- name: context
  description: Product context, audience, constraints, and any known brand assets.
  required: true
- name: stack
  description: "Preferred stack (e.g. nextjs, vite-react). Ask if unclear."
  required: false
---

You are bootstrapping the **{{app_name}}** frontend app using a document-first workflow: **design questions → design doc → user approval →
initialize stack → implement**.

User context:

{{context}}

{{#if stack}} Preferred stack: {{stack}} {{/if}}

Read `knowledge://ronyup/architecture/frontend-design-documents` and
`knowledge://ronyup/architecture/frontend-design-template` first.

Also read these skill files (do not skip — they are required, not optional):

- `.agents/skills/frontend-design/SKILL.md`
- `.agents/skills/design-tokens/SKILL.md`
- `.agents/skills/typography/SKILL.md`

---

## Phase 1 — Clarify (ask before assuming)

1. Confirm frontend topology: one app at `frontend/` or multiple under `frontend/<app>/`.
2. Confirm target app slug: **{{app_name}}** and stack (Next.js, Vite+React, …).
3. Ask design questions the user has not already answered:
   - Aesthetic direction and references (what to avoid: generic AI defaults)
   - Brand constraints vs freedom to propose palette/type
   - Light/dark/both; marketing vs dashboard/data-dense
   - Must-have v1 screens

**Gate:** Do not run framework CLIs or write UI until Phase 2 doc is approved.

---

## Phase 2 — Design document

**Goal:** `docs/design/{{app_name}}-frontend-design.md`

1. Write the document from the template with `status: draft` frontmatter.
2. Include a concrete token plan (colors, type, spacing, signature element) and design-system rules components must follow.
3. Present a short summary and **wait for user approval** (`status: approved` in frontmatter, or explicit user confirmation).

Never set `status: approved` yourself.

---

## Phase 3 — Initialize stack

Only after the design doc is approved:

1. Initialize the app in `frontend/` or `frontend/{{app_name}}/` with the chosen stack.
2. Wire semantic tokens into `globals.css` / Tailwind / shadcn theme per the approved doc.
3. Expose npm scripts required by `frontend/verify.sh`: `typecheck`, `lint`, `build`, `test`, and `build-storybook` once Storybook is added.

---

## Phase 4 — Implement

1. Build v1 surfaces from the design doc; derive every color and type choice from the token plan.
2. Use `shadcn`, `nextjs-modern`, `ux-quality`, and `storybook` skills as needed.
3. Run `bash frontend/verify.sh` (or `make -C frontend verify`) and fix until green before reporting done.

If implementation reveals design gaps, update the design doc (revision history) before or alongside code changes.
