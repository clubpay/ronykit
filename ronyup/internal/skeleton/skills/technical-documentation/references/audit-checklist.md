# Audit Checklist: Rule IDs, Severity, and Scoring

The frozen rule table this skill cites, plus the mechanics that turn it into a score and a ship/no-ship call.

1. [How to use this checklist](#how-to-use-this-checklist)
2. [The rule table](#the-rule-table)
3. [Severity rubric and the shippability gate](#severity-rubric-and-the-shippability-gate)
4. [Scoring mechanics](#scoring-mechanics)
5. [Finding a local style guide first](#finding-a-local-style-guide-first)
6. [Non-English documents](#non-english-documents)
7. [Automating the `[EN]` layer with Vale](#automating-the-en-layer-with-vale)
8. [Worked mini-audit](#worked-mini-audit)

---

## How to use this checklist

Read this file before any audit or improve pass. Rule IDs are stable identifiers, not shorthand you can invent: cite an ID only after reading its row. They fill the Findings column "Rule (ID + name)" and, in improve mode, the change-log column "Rule ID + name".

- One ID per finding; pair them (`P7 / P8`) only when a single edit fixes both — the pair takes the higher severity.
- Severity comes from the table below, not from how annoying the defect felt.
- Explanations and before/after pairs live in the owner file named in the last column; here each rule is one row.

Use the audit output block defined in SKILL.md, section 8, unchanged.

Source: skill rule (SKILL.md, section 8).

---
## The rule table

Sixty rules in seven groups. The `(inferred)` and `(convention)` labels mark rows that are not verbatim Google guidance; keep the label when you cite them. Source slugs are pages under `developers.google.com/style` unless another origin is named.

**R — reader and document type**

| ID | Rule | Severity | [EN] | Source | Owner file |
|----|------|----------|------|--------|-----------|
| R1 | First paragraph states who the doc is for and what they'll accomplish | High | — | tech-writing/one "Documents" | document-types.md |
| R2 | Doc type matches the reader's task: tutorial / how-to / concept / reference / README | High | — | (inferred taxonomy; Google covers procedures + concept-heading style) | document-types.md |
| R3 | Prerequisites listed before the first step ("Before you begin") | High | — | procedures + Google docs practice (convention) | document-types.md |
| R4 | Scope and non-scope stated | Medium | — | tech-writing/one "Documents" | document-types.md |
| R5 | Ends with verification and/or "What's next" | Medium | — | (convention) | document-types.md |
| R6 | Key points first; one idea per paragraph, lead sentence carries it | Medium | — | paragraphs, tech-writing/one "Paragraphs" | document-types.md |

**V — voice**

| ID | Rule | Severity | [EN] | Source | Owner file |
|----|------|----------|------|--------|-----------|
| V1 | Second person "you"; never "we"/"the user" for the reader | Medium | — | person | voice-and-words.md |
| V2 | Active voice unless the actor is unknown or irrelevant | Medium | — | voice | voice-and-words.md |
| V3 | Present tense; "will" only for genuinely later effects | Medium | — | tense | voice-and-words.md |
| V4 | No "please" outside permission/forgiveness | Low | — | tone | voice-and-words.md |
| V5 | No "simply/easily/just/quickly", superlatives, or absolutes | Low | — | excessive-claims | voice-and-words.md |
| V6 | No anthropomorphism (software doesn't want/see/think) | Medium | — | anthropomorphism | voice-and-words.md |
| V7 | Timeless: no "currently/now/new/soon"; no pre-announcing | Medium | — | future, timeless-documentation | voice-and-words.md |
| V8 | Contractions allowed; negations prefer "isn't/don't" | Low | `[EN]` | contractions | voice-and-words.md |
| V9 | Inclusive language (no master/slave, blacklist/whitelist, sanity check, gendered generic pronouns) | Medium | — | inclusive-documentation | voice-and-words.md |
| V10 | Global audience: no idioms, culture-bound examples, "once" meaning "after" | Low | — | translation | voice-and-words.md |

**W — words and sentences**

| ID | Rule | Severity | [EN] | Source | Owner file |
|----|------|----------|------|--------|-----------|
| W1 | Condition or goal before the instruction | Medium | — | sentence-structure | voice-and-words.md |
| W2 | One idea per sentence; short sentences | Low | — | tech-writing/one "Short sentences" | voice-and-words.md |
| W3 | Abbreviations spelled out on first use | Medium | — | abbreviations | voice-and-words.md |
| W4 | No e.g. / i.e. / etc. / vs. | Low | `[EN]` | abbreviations | voice-and-words.md |
| W5 | can / may / might used for ability / permission / possibility | Low | `[EN]` | word-list | voice-and-words.md |
| W6 | Jargon defined or avoided for the stated reader | Medium | — | jargon | voice-and-words.md |
| W7 | Word-list compliance (one rule; entries live in voice-and-words.md) | Low | `[EN]` | word-list | voice-and-words.md |
| W8 | American spelling; serial comma | Low | `[EN]` | highlights, commas | voice-and-words.md |

**S — structure and formatting**

| ID | Rule | Severity | [EN] | Source | Owner file |
|----|------|----------|------|--------|-----------|
| S1 | Sentence case for titles and headings | Low | — | headings | structure-and-formatting.md |
| S2 | Task headings = bare imperative; concept headings = noun phrase; no "-ing" | Medium | — | headings | structure-and-formatting.md |
| S3 | No skipped heading levels; no trailing period; (inferred) no heading immediately followed by a heading | Low | — | headings | structure-and-formatting.md |
| S4 | Lists: complete intro sentence ending in a colon; parallel items | Medium | — | lists | structure-and-formatting.md |
| S5 | Numbered lists only for sequences; bullets otherwise; description lists for pairs | Medium | — | lists | structure-and-formatting.md |
| S6 | Tables: header row, intro sentence, no merged or empty cells | High | — | tables, accessibility | structure-and-formatting.md |
| S7 | Notices: don't stack; Note / Caution / Warning semantics; (inferred) one per section | Medium | — | notices | structure-and-formatting.md |
| S8 | Cross-references use "see"; never above/below | Medium | — | cross-references | structure-and-formatting.md |
| S9 | Link text descriptive; never "click here" / "this link" | High | — | link-text, accessibility | structure-and-formatting.md |
| S10 | Purposeful alt text; information never only in an image | Blocking | — | images, accessibility | structure-and-formatting.md |
| S11 | Unambiguous dates; numbers zero–nine spelled out, 10+ numerals | Low | `[EN]` | dates-times, numbers | structure-and-formatting.md |
| S12 | Bold for UI elements, code font for code, italics sparingly for terms | Low | — | text-formatting | structure-and-formatting.md |

**P — procedures and code**

| ID | Rule | Severity | [EN] | Source | Owner file |
|----|------|----------|------|--------|-----------|
| P1 | Steps numbered, one imperative action per step; a missing required step escalates to Blocking | High | — | procedures | procedures-and-code.md |
| P2 | Single step = bullet; sub-steps a/b/c then i/ii/iii | Low | — | procedures | procedures-and-code.md |
| P3 | Optional steps prefixed "Optional:" | Low | — | procedures | procedures-and-code.md |
| P4 | Each step says where to act and what result to expect | Medium | — | procedures | procedures-and-code.md |
| P5 | UI names bold, casing matches the UI; click / tap / select by device | Low | — | ui-elements | procedures-and-code.md |
| P6 | Code font for filenames, paths, commands, flags, params, values — not product names | Medium | — | code-in-text | procedures-and-code.md |
| P7 | Every code sample introduced by a sentence ending in a colon | Medium | — | code-samples | procedures-and-code.md |
| P8 | Placeholders `ALL_CAPS_WITH_UNDERSCORES`, no MY_/YOUR_, explained after the sample | Medium | — | placeholders | procedures-and-code.md |
| P9 | Command-line syntax: `[optional]`, `{a\|b}`, `...` | Low | — | code-syntax | procedures-and-code.md |
| P10 | Samples runnable, minimal, ≤80-char lines, expected output shown | Medium | — | code-samples, tech-writing/two "Sample code" | procedures-and-code.md |
| P11 | Every command, flag, parameter, and behavior verified in code or given by the user | Blocking | — | (skill rule) | procedures-and-code.md |

**A — API reference, docstrings, CLI help**

| ID | Rule | Severity | [EN] | Source | Owner file |
|----|------|----------|------|--------|-----------|
| A1 | Every exported type / function / method / field / constant documented | High | — | api-reference-comments | api-reference.md |
| A2 | Method descriptions open with the category verb (Gets / Sets / Checks whether / Creates / Returns…); no "This method…" | Medium | — | api-reference-comments | api-reference.md |
| A3 | Non-boolean params start "The…/A…"; booleans "If true, … If false, …" or "True if …; false otherwise" | Medium | — | api-reference-comments | api-reference.md |
| A4 | Return values and exceptions documented ("Thrown when…") | High | — | api-reference-comments | api-reference.md |
| A5 | Deprecated elements name the replacement in the first sentence | High | — | api-reference-comments | api-reference.md |
| A6 | CLI help: usage line, one-line synopsis, every flag described, placeholders in caps | Medium | — | code-syntax + (convention — no Google `--help` page) | api-reference.md |
| A7 | Reference text is descriptive third-person present; guides are imperative | Low | — | reference-verbs | api-reference.md |

**N — release notes, changelogs, migration**

| ID | Rule | Severity | [EN] | Source | Owner file |
|----|------|----------|------|--------|-----------|
| N1 | Newest version first; `Unreleased` section; ISO dates in version headings | Medium | — | keepachangelog.com | release-notes.md |
| N2 | Entries grouped Added / Changed / Deprecated / Removed / Fixed / Security | Medium | — | keepachangelog.com | release-notes.md |
| N3 | Entry = what changed + reader impact + action; never raw commit messages | High | — | keepachangelog.com | release-notes.md |
| N4 | Breaking changes called out first with a link to migration steps | High | — | (convention) | release-notes.md |
| N5 | Deprecation entries name the replacement and removal version/date | High | — | keepachangelog.com, api-reference-comments | release-notes.md |
| N6 | Migration guide is a procedure with before/after snippets and a verify step | High | — | procedures (applied) | release-notes.md |

Source: Google Developer Documentation Style Guide (slug per row), Google Technical Writing One and Two, keepachangelog.com for N1–N5; labelled rows are this skill's own.

---

## Severity rubric and the shippability gate

Severity answers what breaks for the reader, not how many words change. Each rule carries its own severity above.

| Severity | Definition |
|----------|-----------|
| Blocking | Facts wrong or unverifiable; a procedure that can't be completed; information carried only by an image, or an image with no alt text |
| High | Wrong doc type, missing audience or prerequisites, steps buried in paragraphs, vague link text, headerless tables, undocumented API elements, changelog entries with no reader impact |
| Medium | Passive voice, future tense, "we"/"the user", anthropomorphism, time-bound wording, unexplained abbreviations, misused notices, missing intro sentences, placeholder style |
| Low (mostly `[EN]`) | Word list, serial comma, contractions, spelling, UI-label casing, Latin abbreviations, numbers and dates |

Blocking is a gate, not a deduction: one Blocking finding makes a doc `Shippable: no` at 9/10, and a doc with none is shippable at 6/10 — separate report lines.

Work Blocking → High → Medium → Low, and within that, structure before voice before words: sentences polished in a section you then delete are wasted, and word edits made before a restructure get made twice.

Source: skill rule — severity rubric and protocol step 5 in SKILL.md, section 8.

---

## Scoring mechanics

Score = Quick Diagnostic rows passed: ten rows, one point each, no partial credit — a row passes only if the whole document satisfies it; a row with no applicable content passes (a concept page with no procedures passes rows 5 and 6), and A- and N-rule findings affect `Shippable` and the Rewrite plan, not the score. Row 10 auto-passes for a non-English doc. Name the band from SKILL.md's Scoring section; don't restate it. Findings cite the rule ID, not the row number:

| Row | Checks | Rule IDs |
|-----|--------|----------|
| 1 | Audience and outcome up front | R1 / R4 |
| 2 | Doc type fits the task | R2 |
| 3 | Facts verified against code | P11 |
| 4 | Headings scan as a contents list | S1 / S2 |
| 5 | Numbered imperative steps, condition first | P1 / W1 |
| 6 | Samples introduced, placeholders explained | P7 / P8 |
| 7 | Second person, active, present, no filler | V1–V6 |
| 8 | Links, cross-references, images, tables accessible | S6 / S8 / S9 / S10 |
| 9 | Timeless | V7 |
| 10 | `[EN]` word list, serial comma, contractions, spelling | W7 / W8 / V8 |

The `**Diagnostic:**` line repeats the number and names the failures: `N/10 — failed rows: <n> (<one-line reason>), …`, one clause per failed row phrased as the observed defect; passing rows aren't listed.

Source: skill rule (SKILL.md, Scoring and Quick Diagnostic).

---

## Finding a local style guide first

A project's conventions outrank Google's; find them before filing a finding against a deliberate choice.

| Look for | Usually settles |
|----------|-----------------|
| `CONTRIBUTING.md` | Docs workflow, tone, required sections |
| `STYLE.md`, `STYLEGUIDE.md`, `docs/style-guide.md` | Explicit house style |
| `docs/contributing/*.md` | Per-docs-set conventions |
| `.vale.ini` | Linter package and word list in force |
| `.markdownlint*` | Heading levels, line length, list markers |
| `.github/PULL_REQUEST_TEMPLATE.md` | The docs checklist the team applies |

When no file states the rules, the docs still do:

- `grep -rin "log in\|sign in" docs/` — the winner by count is the house term.
- Heading case: sample a dozen `^#{1,3} ` lines; title case throughout is a convention, not twelve findings.
- Placeholders: `grep -rn "YOUR_\|<[a-z-]*>" docs/` — a consistent house form beats P8.
- Notices: match the toolchain's syntax (`> **Note**`, `:::note`, `{: .note }`).

Precedence: project guide → Google → Merriam-Webster (spelling), Chicago (general), Microsoft Writing Style Guide (technical). A convention wins in its scope only if it is consistent; an inconsistent habit is a finding.

Record it on the report's `**Local style guide:**` line: the file and that you honored it, any rule you deferred (`STYLE.md — title-case headings, S1 not applied`), or `none — Google applies`.

Source: developers.google.com/style/about for precedence; the file list and grep probes are convention.

---

## Non-English documents

Structural and content rules apply unchanged: all of R, S1–S10 and S12, P1–P11, A1–A6, N1–N6, plus V1, V2, V3, V6, V7, V9 and W1, W3, W6. A Japanese how-to still needs an audience statement, numbered steps, and verified flags.

Skip the `[EN]` rules — V8, W4, W5, W7, W8, S11 — which encode English spelling, word choice, and punctuation. V4, V5, V10, W2, S1, S2, and A7 sit in between: the intent transfers, the exact test doesn't.

Record the language and the skipped layer on the report's `**Language:**` line: `de — [EN] rules skipped (V8, W4, W5, W7, W8, S11)`. Row 10 then auto-passes, out of the same 10. For a mixed-language doc, apply the `[EN]` rows to the English passages only and record `mixed (en/pl) — [EN] applied to English text`.

Never translate a document unless asked: it hides the page from its readers and turns a style audit into a content change.

Source: skill rule (SKILL.md, Core Principle) with translation and global-audience guidance.

---

## Automating the `[EN]` layer with Vale

Vale with the Google package catches most Low-severity English findings mechanically. A minimal configuration at the repo root:

```ini
StylesPath = .vale/styles
MinAlertLevel = suggestion
Packages = Google

[*.md]
BasedOnStyles = Vale, Google
```

Run `vale sync` to fetch the package into `.vale/styles`, then `vale docs/`.

Coverage, mapped to these IDs (inferred — the package's rule names are Google's, not this skill's): roughly V2–V5, V8, W4, W7, W8, and S1 — passive voice, "will", "please", "simply/just", contractions, e.g./i.e., the word list, spelling and serial comma, exclamation marks, heading case.

Out of reach, so still a human pass: every R, A, and N rule, P11, plus S6 table semantics, S10 image-only information, P1 step decomposition. A green run is evidence for row 10, not a passing audit.

Source: Vale — errata-ai/Google package.

---

## Worked mini-audit

A README excerpt for `shipd`, a fictional deploy CLI:

```markdown
# Getting Started With Shipd

Shipd is the fastest way to deploy your app. Currently, we are excited to
announce that our new dashboard will be shown once the user has logged in.

## Installing The CLI

Please simply run the install script below. After that you should click on the
Deploy button in order to deploy, and the status will be changed. For
configuration options, click here.

`curl -sL get.shipd.io | sh && shipd deploy --key=<your-key> --turbo`

| shipd deploy | shipd rollback |
| --- | --- |
```

Report header (full template in SKILL.md, section 8):

```
**Score:** 1/10 — ≤4 band   **Shippable:** no (Blocking below)
**Diagnostic:** 1/10 — failed rows: 1 (no audience or outcome), 3 (`--turbo`
unverifiable), 4 (title case, "-ing"), 5 (steps in prose), 6 (no intro,
`<your-key>`), 7 ("we", passive, "please", "fastest"), 8 ("click here",
headerless table), 9 ("Currently", pre-announced dashboard), 10 ("log in",
"in order to")
```

**Findings:**

| # | Location | Rule (ID + name) | Before | After | Severity |
|---|----------|------------------|--------|-------|----------|
| 1 | Headings | S1 sentence case; S2 imperative | "Getting Started With Shipd" | "Get started with Shipd" | Medium |
| 2 | Intro | V7 timeless; V1 second person | "Currently, we are excited to announce…" | "After you sign in, the dashboard appears." | Medium |
| 3 | Install paragraph | V4 no please; V5 no simply | "Please simply run the install script" | "To install the CLI, run:" | Low |
| 4 | Install paragraph | P1 numbered steps; W1 condition first | "you should click on the Deploy button" | "1. Click **Deploy**. The status changes to **Deploying**." | High |
| 5 | Command line | P7 colon intro; P8 placeholders | `--key=<your-key>`, no intro | `--key=API_KEY`, explained after the sample | Medium |
| 6 | `--turbo` flag | P11 facts verified in code | In no source file or help output | `TODO(verify): --turbo not in cmd/deploy.go` | **Blocking** |
| 7 | "click here" | S9 descriptive link text | "For configuration options, click here." | "see Configure Shipd" | High |
| 8 | Command table | S6 header row and intro | Commands used as the header row | Intro sentence + `Command` / `Description` header | High |

**Score.** Row 2 passes — a README is the right type for a first deploy; the other nine fail as the Diagnostic line lists. That is **1/10**, the `≤4` band: rewrite from the README skeleton; `Shippable: no` until finding 6 resolves.

Source: skill rule — worked example applying the rule IDs above.
