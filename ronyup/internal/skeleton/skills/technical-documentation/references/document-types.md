# Document Types

Deepens SKILL.md §1 (Know the Reader and the Document's Job): which document type fits a given reader's question, the audience and scope statements that open a page, prerequisite and closing patterns, skeletons for the five common page types, paragraph structure, and the self-editing pass for a full doc set.

## Choose the Document Type

Every page answers one reader's question. Match the type to the question before writing a word — mixing types on one page forces the reader to read everything to find their one task.

| Type | Reader's question | Signals it's the right type | Must not contain |
|---|---|---|---|
| Tutorial | "Walk me through building this, start to finish?" | Reader is new to the feature and wants one guaranteed win | Exhaustive option lists; alternate paths; unexplained jargon |
| How-to | "How do I do this one task?" | Reader already has a working setup and a specific goal | Onboarding scaffolding; the "why" behind the feature |
| Concept | "What is this, and why does it work this way?" | Reader wants a mental model before acting | Numbered steps; command output; UI clicks |
| Reference | "What are the exact parameters, fields, or flags?" | Reader already knows what they want and needs the exact facts | Narrative explanation; opinion; "why" content |
| README / getting started | "What is this, and how do I start?" | Reader is arriving for the first time, evaluating or installing | Deep conceptual explanations; exhaustive configuration matrices |

**One task per page (R2).** A page that answers two of these questions answers neither well. When a draft mixes types — a concept paragraph, then steps, then a reference table — split it along these lines rather than adding more headings to the same page. This taxonomy isn't in Google's guide itself; it's inferred from the guide's Procedures page and its concept-heading guidance (S2, structure-and-formatting.md), generalized to the other common developer-doc shapes.

Source: procedures, headings (inferred taxonomy — R2)

## Audience and Scope Statements

A page's first paragraph does two jobs before any instruction starts: tell the reader whether they're on the right page, and tell them what they'll be able to do afterward (R1). A scope and non-scope statement extends this by naming what's deliberately left out, with a link to where that content lives (R4).

**Audience statement**
- Before: "This page describes the notification system."
- After: "This page is for backend developers adding Shipit webhook notifications to an existing service. It assumes you can deploy an HTTP endpoint and have admin access to a Shipit project."

**Scope and non-scope statement**
- Before: "This guide covers webhooks."
- After: "This guide shows how to subscribe to deployment events and verify webhook signatures. It doesn't cover configuring retry policies or building a custom event bus — see Webhook retry policies."

**"What you'll do" intro**
- Before: "Shipit is a deployment platform that many teams use to ship software faster and reduce risk across environments."
- After: "By the end of this page, you can configure Shipit to redeploy your app automatically whenever a build passes CI."

The non-scope sentence is what rescues a reader who followed a search result to the wrong page — without it, they read the whole doc before discovering it doesn't cover their case.

Source: tech-writing/one "Documents" (R1, R4)

## Before You Begin and What's Next

**Before you begin (R3)** lists everything the reader needs before step 1 — nothing they'll discover along the way. It belongs there, not folded into step 4.

Belongs in prerequisites:
- Software and minimum versions ("Shipit CLI 4.2 or later")
- Required roles or permissions ("Admin role on the target project")
- Tools or SDKs that must already be installed
- Accounts, API keys, or a specific starting state ("A GitHub repository connected to Shipit")

Doesn't belong: conceptual background (link a concept page instead), and setup steps that belong to a different, already-linked page.

- Before: "4. Before running this command, make sure you've installed the Shipit CLI and have admin access on the project."
- After:
  ```
  ## Before you begin

  - Shipit CLI 4.2 or later
  - Admin role on the target project
  - A GitHub repository connected to Shipit
  ```

**Verification and What's next (R5).** A procedure that ends on the last action leaves the reader guessing whether it worked. Close with a verification step phrased as a check, and the expected output, then point to the next task.

- Before: "4. Run `shipit deploy`. That's it — you're done."
- After:
  ````
  Confirm that the deployment succeeded:

  ```
  shipit status my-app
  ```

  The command prints `Status: healthy`.

  ## What's next

  - Configure automatic rollbacks
  - Send deployment events to Slack
  ````

The exact headings "Before you begin" and "What's next", and the "Confirm that…" verification phrasing, aren't quoted rules in Google's guide — they're the convention this skill standardizes on, built from the guide's Procedures page and Google's own documentation practice.

Source: procedures; tech-writing/one "Documents" (R3, R5; heading wording and verification phrasing are convention)

## Skeletons (convention)

None of these page shapes are named in Google's guide — they're this skill's convention for applying R1-R6 consistently. Use them as a starting outline, not a rigid template.

**README**
1. Project name and one-line purpose — what it does, in one sentence
2. Who it's for — the audience statement
3. Quick start — three steps: install, configure, run
4. Links out — full docs, contributing guide, license

**Getting started (docs-site landing page)**
1. Title
2. What you'll do — the outcome statement (R1)
3. Before you begin — prerequisites (R3)
4. Install, first command, verify
5. What's next (R5)

**Tutorial**
1. Title stating the goal ("Build a webhook listener with Shipit")
2. What you'll build — one or two sentences, the end state
3. Before you begin — prerequisites (R3)
4. Numbered steps, each with the expected result stated after the action
5. Clean up — remove anything created only for the tutorial
6. What's next (R5)

**How-to**
1. Title as a bare imperative task ("Configure a webhook endpoint") — heading form is S2 (structure-and-formatting.md)
2. One sentence of context, only if more than one approach exists
3. Before you begin — only what's specific to this task, not full onboarding
4. Numbered steps, condition before instruction
5. A short verification step (R5)

**Concept**
1. Title as a noun phrase ("Webhook signing") — heading form is S2 (structure-and-formatting.md)
2. What it is — key point first (R6)
3. Why it works this way — the mechanism or trade-off
4. How it fits with other pieces
5. A link to the procedure that acts on this concept

Source: convention

## Paragraphs and Key Points First

A paragraph carries one idea. Its lead sentence states that idea; everything after supports it. Aim for three to five sentences, and put the information the reader needs before the background that explains it (R6).

**Key information before background**
- Before: "Shipit's build system uses a layered cache to speed up repeated builds, an approach the team added after users on large monorepos reported slow CI times, and it now applies to every project by default."
- After: "Shipit caches build layers by default, so repeated builds finish faster. The team added this after users on large monorepos reported slow CI times."

**One idea per paragraph**
- Before: "Deployments run in isolated containers, so one project's build never affects another. You can also configure a custom domain for each environment, which requires DNS access and a verified certificate before Shipit will route traffic to it."
- After:
  ```
  Deployments run in isolated containers, so one project's build never
  affects another.

  You can configure a custom domain for each environment. This requires
  DNS access and a verified certificate before Shipit routes traffic to it.
  ```

A reader skimming only lead sentences should still get the page's argument — that's the test for whether the point comes first.

Source: tech-writing/one "Paragraphs" (R6)

## Self-Editing and Organizing Large Doc Sets

A single page needs a different check than a doc set spanning dozens of pages. Apply both.

**Self-editing a page:**

| Technique | What to do | Catches |
|---|---|---|
| Adopt the reader's persona | Reread as the stated audience, not as the author who already knows the system | Missing prerequisites, unexplained jargon |
| Read it aloud | Read every sentence out loud, one at a time | Run-ons, missing words, buried passive voice |
| Come back later | Leave a draft for a few hours or a day before the final pass | Errors invisible to a tired eye |
| Run the checklist | Walk the page against the Quick Diagnostic rows in SKILL.md | Systemic gaps a single read-through misses |
| Find a peer editor | Have someone unfamiliar with the feature follow the doc | Steps that only work because the author knows a hidden step |

**Organizing a doc set:**
- Outline first. Write the heading structure before any prose, and get it approved before filling it in.
- State the set's purpose up front. The landing page's first sentence says what the set covers, the same way a single page's first paragraph does (R1).
- Headings as a table of contents. A reader who reads only the headings across the set should be able to tell what's where and pick the right page.
- Keep tutorials and how-tos on separate pages. A tutorial's teaching narrative and a how-to's task focus interfere with each other on the same page — this is R2's one-task-per-page rule applied across an entire set, not just one page.

Source: tech-writing/two "Self-editing" and "Organizing large documents"

## Worked Example

A mixed page interleaves concept, steps, and reference. Here's one before a split:

```markdown
# Webhooks

Shipit webhooks let external services react to deployment events without
polling. Every payload is signed with an HMAC-SHA256 digest so subscribers
can verify the sender before trusting the contents; this became necessary
once webhook endpoints started appearing on the public internet, where
anyone could send forged events.

1. Open your project settings and click **Webhooks**.
2. Click **Add endpoint** and enter your HTTPS URL.
3. Copy the signing secret shown after you save.

The `event` field is one of `deploy.started`, `deploy.succeeded`, or
`deploy.failed`. The `signature` header holds the HMAC digest, and
`timestamp` is a Unix epoch integer used to reject replayed requests
older than five minutes.

4. Deploy your project and check your endpoint's logs for a request.
```

A reader who only wants to configure an endpoint has to read a security rationale and a payload reference to find steps 1 through 4, scattered across the page. Splitting by task (R2) gives two linked pages — headings only, since the prose above already carries the content:

**Page 1 — Concept: "Webhook signing"**
1. Webhook signing
2. Why Shipit signs every payload
3. Payload fields (`event`, `signature`, `timestamp`) — reference material about the payload sits with the concept, not the procedure
4. See "Configure a webhook endpoint" to set one up

**Page 2 — How-to: "Configure a webhook endpoint"**
1. Configure a webhook endpoint
2. Before you begin — an HTTPS-reachable endpoint, admin role on the project
3. Numbered steps 1-3 (unchanged)
4. Confirm that your endpoint received a request — the stranded step 4 becomes a verification step, with the expected log line stated
5. What's next

Source: convention (applies R1-R6 together)
