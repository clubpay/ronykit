# Structure and Formatting

1. [Headings (S1, S2, S3)](#headings-s1-s2-s3)
2. [Lists (S4, S5)](#lists-s4-s5)
3. [Tables (S6)](#tables-s6)
4. [Notices (S7)](#notices-s7)
5. [Cross-references (S8)](#cross-references-s8)
6. [Link text (S9)](#link-text-s9)
7. [Images and alt text (S10)](#images-and-alt-text-s10)
8. [Numbers, dates, and units (S11)](#numbers-dates-and-units-s11-en)
9. [Text formatting summary (S12)](#text-formatting-summary-s12)
10. [Worked example](#worked-example)

This file deepens Framework section 4 (Structure: Headings, Lists, Tables, Notices) with the full rule text, more before/after pairs, and the numbers, dates, and text-formatting rules that section only summarizes. It owns **S1–S12**. Code-font specifics, placeholders, and UI-element bold live in `procedures-and-code.md` (P6, P8, P5); paragraph rules live in `document-types.md` (R6).

## Headings (S1, S2, S3)

Headings are the reader's table of contents before they read a word of body text — a reader scans the heading list, finds their task, and jumps there. A heading that doesn't say what the section does, or that breaks case or level conventions, breaks the scan.

### S1 — Sentence case

Capitalize the first word and proper nouns only; leave common nouns lowercase even when a UI element shows them capitalized. Proper nouns, product and brand names, and code stay exactly as written: Google Cloud, Cascade, `README.md`.

- "Configuring The Database Connection" → "Configure the database connection"
- "API Reference For The Webhooks Endpoint" → "API reference for the Webhooks endpoint"

### S2 — Task headings vs. concept headings

A task heading is a bare imperative — the verb the reader performs: "Create a webhook". A concept heading is a noun phrase — the thing the reader is learning about: "Webhook lifecycle". Neither ever ends in "-ing"; an "-ing" heading hides which of the two it is.

- "Creating a Webhook" (task, disguised as a gerund) → "Create a webhook"
- "Troubleshooting Deploy Failures" (task) → "Troubleshoot deploy failures"
- "Webhook Payload Format" (concept, wrong case) → "Webhook payload format"

### S3 — Levels, endings, and adjacency

Don't skip heading levels (H2 straight to H4 with no H3 in between) — screen readers announce level jumps as broken structure. Headings never end with a period. And (inferred) avoid a heading immediately followed by another heading with no body text between them — a reader lands on the child heading with no idea what the parent section covers.

**Full outline, before:**

```markdown
# Cascade CLI Reference.

### Installing Cascade
#### Prerequisites
##### Supported platforms
### Deploying An App
### Troubleshooting Common Errors
```

Problems: trailing period on the title; the outline jumps from H1 to H3 (no H2); "Prerequisites" is immediately followed by "Supported platforms" with nothing said in between; every heading is "-ing" and title case.

**After:**

```markdown
# Cascade CLI reference

## Install Cascade

### Prerequisites

Before you install Cascade, confirm you have Node.js 18 or later.

### Supported platforms

## Deploy an app

## Troubleshoot common errors
```

Source: headings

## Lists (S4, S5)

### S4 — Intro sentences and parallelism

Every list needs an intro sentence that is grammatically complete on its own, ending in a colon. A sentence that only works once you mentally append the first bullet — "Use the `--format` flag to:" — breaks the moment an item doesn't start with a verb. Every item in a list takes the same grammatical form as the others: all imperatives, all nouns, all noun phrases.

**Before (fragment intro, mixed forms):**

```markdown
Use the --format flag to:
- Print JSON
- Print YAML
- Print a table
```

**After (complete intro, parallel nouns):**

```markdown
The --format flag supports the following output types:
- JSON
- YAML
- Table
```

**Before (non-parallel steps):**

```markdown
Before you deploy, complete these steps:
- Installing the CLI
- You need an API key
- Configure the project
```

**After:**

```markdown
Before you deploy, complete these steps:
- Install the CLI
- Get an API key
- Configure the project
```

### S5 — Numbered vs. bulleted vs. description lists

Number a list only when order matters — the reader must do item 1 before item 2. Everything else is bulleted, including options, flags, and requirements that stand independently of each other.

**Before (numbered, but the items aren't a sequence):**

```markdown
1. Enable verbose logging
2. Enable colorized output
3. Enable strict mode
```

**After:**

```markdown
- Enable verbose logging
- Enable colorized output
- Enable strict mode
```

For term-and-definition pairs, a description list beats a two-column table — the term reads as a heading, not a table cell, and there's no header row to skip. Markdown has no universal native syntax for this, but the widely supported Markdown Extra / Pandoc form is:

```markdown
`--dry-run`
:   Previews changes without applying them.

`--verbose`
:   Prints each step as it runs.
```

**Punctuation** `[EN]` (inferred): capitalize the first word of every item regardless of form. A complete sentence ends with a period; a fragment doesn't.

- Fragment, no period: "- Verbose output"
- Complete sentence, period: "- Verbose output is enabled by default."

Source: lists

## Tables (S6)

A table needs a header row and an intro sentence that states what it lists — "The following table lists the `cascade deploy` flags:" — so a reader (and a screen reader) knows what they're about to scan before they hit the grid. Never merge cells and never leave one empty: an empty cell reads to assistive tech as if nothing is there at all, not as "not applicable". Write "None" or "Not applicable" instead (convention). Keep cell phrasing parallel down a column — if one description is a sentence, they all are.

Reach for a bulleted or description list instead of a table when there's only one dimension of comparison (a plain list of flags with no second attribute) or when most cells would carry long prose — tables earn their header row only when the data is genuinely tabular across two or more attributes.

**Before:**

```markdown
| Flag | |
|------|--|
| --verbose | |
| --dry-run | Preview changes |
```

**After:**

```markdown
The following table lists the `cascade deploy` flags:

| Flag | Description |
|------|-------------|
| `--verbose` | Prints each step as it runs. |
| `--dry-run` | Previews changes without applying them. |
```

Source: tables, accessibility

## Notices (S7)

A notice interrupts the reader's flow, so it earns that interruption only when the information changes what they do next.

- **Note** — useful, optional information the reader can act on or skip: "Cascade caches build artifacts in `.cascade/cache`. Delete this directory to force a clean build."
- **Caution** — proceed carefully; the mistake is recoverable but costly: "Changing the region after deployment migrates your database and can take up to 30 minutes."
- **Warning** — serious harm or irreversible loss: "Running `cascade reset --hard` permanently deletes all environments and can't be undone."

Don't stack them. (Inferred) one notice per section is a practical ceiling — three boxes in a row train the reader to skip all three, including the one that matters.

Fold a notice into body text whenever it doesn't change the reader's next action — a plain sentence in the paragraph carries the same information without the visual interruption.

**Before:**

```markdown
Run `cascade deploy` to publish your app.

> **Note:** This command requires an active internet connection.
```

**After:**

```markdown
Run `cascade deploy` to publish your app. This command requires an internet connection.
```

Source: notices

## Cross-references (S8)

Point to other content with "see", never "refer to" or "check out". Never say "above" or "below" — pages reflow, get translated into languages that reorder content, and get read out of order by screen readers and search results. Use "the following" for what comes immediately after in the same page, "the preceding" for what came immediately before, and a named link for anything farther away.

Link when the referenced material lives on another page, or is optional depth the current task doesn't require. Inline the fact instead — repeat the one sentence the reader needs — when sending them away would interrupt a procedure they're mid-way through.

- "For more info, check out the docs above." → "For more information, see Configure authentication."
- "As mentioned below, you'll need an API key." → "You need an API key; see Get an API key."
- "Refer to the table above for exit codes." → "See the preceding table for exit codes."

Source: cross-references

## Link text (S9)

Link text should read as the destination's title, or a close description of it, so it makes sense pulled out of the sentence — many screen readers list a page's links with no surrounding text at all. Never link "click here", "this link", or a bare pasted URL. Put the link on the words that name the target, not on a filler verb.

- "To learn about rate limits, click here." → "For rate limit details, see [Rate limits](#)."
- "Read more at https://docs.cascade.dev/webhooks." → "For webhook payload formats, see [Webhook payloads](#)."
- "This link explains how authentication works." → "See [How authentication works](#) for the full flow."

Source: link-text, accessibility

## Images and alt text (S10)

Alt text states what the image communicates in this context, not what it literally shows — skip "Image of" and "Screenshot of"; the screen reader already announces that it's an image. Keep it to one concise phrase or sentence, not a full transcription of every pixel.

Information must never live only in an image: if a diagram is the sole place a required value, flag, or step appears, the doc is broken for anyone who can't see it and for anyone who needs to copy that value. This is a shippability gate (Blocking) — put the same fact in the surrounding text.

Use a screenshot to show where something sits in a UI or to confirm a visual result — a filled-in form, a chart the reader compares theirs against. Use text or a code block for anything the reader types, copies, or runs: text is copyable, searchable, and translates; a screenshot of a command is none of those. Write the caption first — naming the figure's one idea before you draw or crop it keeps the image on message (tech-writing/two) — and never let the caption alone carry a detail that's missing from the alt text.

**Before:**

```markdown
![Screenshot of the dashboard](dashboard.png)

Click the button to deploy.
```

**After:**

```markdown
![The Deploy button in the top-right corner of the Cascade dashboard, next to the environment selector](dashboard.png)

Click **Deploy** in the top-right corner of the dashboard.
```

Source: images, accessibility

## Numbers, dates, and units (S11) `[EN]`

Spell out zero through nine; use numerals for 10 and up. Always use numerals with units, versions, and measurements, no matter how small the number: "3 MB", "version 2 of the API", "5 retries". Dates must be unambiguous — "2026-08-29" or "August 29, 2026" — never "08/29/26" or "29/08/26", which read as different dates depending on the reader's locale. Include a time zone whenever a time matters across regions: "2:00 PM UTC", not "2:00 PM". Units get a space and the standard symbol, never a spelled-out or invented abbreviation: "10 MB", not "10MB" or "10 megs". The time-zone and unit-spacing forms are inferred — the sourced pages cover number and date formats.

- "The free tier includes 3 projects and up to 100mb of storage." → "The free tier includes three projects and up to 100 MB of storage."
- "The migration finished on 08/09/26 at 2pm." → "The migration finished on 2026-08-09 at 2:00 PM UTC."
- "This feature requires SDK version two." → "This feature requires SDK version 2."

Source: numbers, dates-times

## Text formatting summary (S12)

| Format | Use for | Example |
|--------|---------|---------|
| **Bold** | UI element names, matching on-screen casing (P5) | Click **Deploy**. |
| `Code font` | Commands, flags, filenames, code, values (P6) | Run `cascade deploy --dry-run`. |
| *Italics* | A new term on its first use; emphasis, used sparingly | A *webhook* is an HTTP callback that Cascade sends when an event occurs. |

Don't format product names — plain text, matching the vendor's own capitalization (P6). See `procedures-and-code.md` for the full code-font, placeholder, and UI-element rules behind P5 and P6.

Source: text-formatting

## Worked example

**Before** — headings break case and nest wrong, the list has a fragment intro and non-parallel items, the table has no intro sentence and empty cells, and two notices stack back to back:

```markdown
# Configuring Webhooks.

## Setting Up

To set up webhooks you can:
- signing secret
- Choose a delivery URL
- pick which events to send

## The Payload

Below is a table of fields you might get back:

| Field | |
|-------|--|
| event | |
| id | The event's ID |

Note: Webhook retries happen automatically.
Caution: If your endpoint returns a non-2xx status the delivery is marked failed and won't retry.
```

**After** — sentence-case imperative headings, one parallel bulleted list with a complete intro, a table with a header row and no empty cells, the routine fact folded into body text, and a single notice:

```markdown
# Configure webhooks

## Set up a webhook

To set up a webhook, complete these steps:

- Generate a signing secret.
- Choose a delivery URL.
- Select which events to send.

## Webhook payload fields

The following table lists the fields in every webhook payload:

| Field | Description |
|-------|-------------|
| `event` | The event type, for example `payment.succeeded`. |
| `id` | The event's unique ID. |

Cascade retries a failed delivery up to five times before it gives up.

**Caution:** Changing the delivery URL cancels any retries already queued for the old URL.
```
