# Elephant–Goldfish Model for RonyKit

Use the Elephant–Goldfish Model (EGM) to design with project context, review
without conversational bias, and leave architectural decisions with the user.
EGM is a spec-driven workflow, not a different language model or a replacement
for tests, code review, or human approval.

Adapted from
[`google/elephant-goldfish-model`](https://github.com/google/elephant-goldfish-model)
at commit `403d8881d62022490637d740657b33f01e321ea8` (Apache-2.0). The
reviewer instructions and rubrics are in
[egm-reviewers.md](egm-reviewers.md); the license is in
[egm-LICENSE](egm-LICENSE).

## When to use

Use EGM for new features, architectural changes, SRS/SDD and frontend design
documents, and multi-file implementation plans. Brainstorming is optional when
requirements and direction are already clear. A small, clear bug fix can use the
normal debugging workflow, but any applicable RonyKit design-document gate
still applies.

During brainstorming, drafting, review, and resolution, change only design and
review artifacts. Do not scaffold or edit production code until the user has
approved the required design documents.

## Roles and isolation

- **Elephant:** the primary agent. It holds project history, asks questions,
  reads RonyKit MCP knowledge, drafts documents, synthesizes findings, and
  applies resolutions approved by the user.
- **Goldfish:** a fresh, read-only subagent created for one reviewer lens in one
  stage. It receives the target document, its referenced source material, the
  assigned lens and rubric, and any explicitly supplied baseline context. It
  does not receive earlier chat, the Elephant's private rationale, or other
  reviewer output unless the stage explicitly requires a prior report.
- **User:** chooses between defensible architectural trade-offs, decides which
  findings to fix or accept, and approves the design.

A same-thread self-review is not a Goldfish review. Check the actual subagent
capability; do not infer it from the host product's name. If fresh delegation is
unavailable, record every requested reviewer as `not run`, stop the EGM review,
and ask whether the user wants a human-only review. Never substitute an
in-thread review and call it EGM.

Use direct, parallel dispatch when the host cannot nest subagents. The Elephant
then deduplicates findings and writes the report instead of using a synthesizer
subagent. Start a new subagent for every lens and every stage; do not resume a
prior reviewer.

Each delegation prompt must include:

1. The absolute path to `references/egm-reviewers.md` and the assigned lens.
2. The absolute target path, line range or semantic section, and rubric.
3. Paths to referenced source files and optional baseline context.
4. Instructions to stay read-only, inspect the actual files, and return findings
   with severity, location, rationale, and a concrete recommendation.
5. No drafting conversation or implied decisions that are absent from the
   supplied artifacts.

Retry a failed or empty reviewer once. If it still does not produce a report,
mark it `not run`; never count it as a reviewer with zero findings. A report
with a `not run` reviewer is incomplete.

## Artifacts and interaction

Keep long drafts, briefs, and reports in files. In chat, provide a short summary,
the artifact path, and the next decision. Ask one question at a time and wait
for the answer at explicit checkpoints.

Unless the repository defines another artifact location, use:

| Artifact | Path |
| -------- | ---- |
| Optional concept brief | `docs/design/<feature>-concept.md` |
| SRS | `docs/design/<feature>-srs.md` |
| SDD | `docs/design/<feature>-sdd.md` |
| Frontend design | `docs/design/<app>-frontend-design.md` |
| Implementation plan | `docs/plans/YYYY-MM-DD-<feature>.md` |
| Review reports | `docs/design/reviews/<document-slug>/` |
| Drift report | `docs/design/reviews/YYYY-MM-DD-drift-report.md` |

Review reports are evidence, not approval. Write design documents with
`status: draft`; only the user approves them, directly or by explicitly
instructing the agent to change the status.

## 1. Brainstorm when the direction needs stress-testing

1. Clarify the problem, users, constraints, current behavior, success measure,
   and non-goals through open-ended questions, one per turn.
2. State the anchored hypothesis in two or three sentences. Get explicit user
   confirmation before launching reviewers.
3. Launch six fresh Goldfish reviewers in parallel using the brainstorm lenses
   in `egm-reviewers.md`: design/cost, codebase/operations, simplicity/reuse,
   UX/comprehension, migration/readiness, and structural maintainability.
4. Deduplicate their challenges and write the concept brief with: overview,
   analysis, goals and non-goals, alternatives weighed with rejection reasons,
   key decisions, and open risks.
5. Ask the user to resolve the objections and approve the brief before using it
   as input to an SRS or SDD. Record the user's decisions; do not silently drop
   rejected alternatives.

Skip this phase when the user has already supplied a clear requirements or
design artifact.

## 2. Draft with RonyKit as Elephant

Use the existing MCP prompts and templates as the structural source of truth:

- Backend: `write-srs` or `design-new-service`, then `write-sdd` after SRS
  approval.
- Frontend: `design-frontend` after topology and aesthetic questions.
- Implementation sequencing: `writing-plans`, derived from the approved SDD.

Read the referenced code and MCP resources instead of relying on memory. Keep
the RonyKit document paths and frontmatter; do not replace them with upstream
EGM paths or templates. When more than one architectural answer is defensible,
present up to three options with mechanisms, trade-offs, and failure conditions,
then let the user choose.

## 3. Review before requesting approval

Run stages in order. Within a stage, run independent reviewers in parallel.
Use a consolidated report unless the user asks for interactive checkpoints after
each stage.

| Stage | Goldfish reviewers and rubrics | Scope |
| ----- | ------------------------------ | ----- |
| 1. Initial zoom out | `zoom-out` with the zoom-out rubric | Whole document; root problem, simplicity, reuse, altitude |
| 2. Comprehension | `comprehension`; no extra rubric | Whole document; include both required paraphrases |
| 3. Design critique | `design` and `structural` with the feature-spec rubric; `style` with project style | Architecture and requirements, not implementation minutiae |
| 4. Intermediate zoom out | a new `zoom-out` reviewer with the zoom-out rubric | Current whole document plus the stage-1 report; focus on additions and mitigations |
| 5. Implementation readiness | `implementation-readiness` and `codebase-details` with the readiness rubric; `style` with project style | Detailed design or plan, exact files, interfaces, migration, and tests |

Select the applicable pipeline:

- **SRS:** stages 1 and 2. Requirements review must not invent a design; design
  alternatives and readiness belong to the SDD.
- **SDD:** all five stages. Stage 5 covers the API, domain, app, repository,
  wiring, migration, and testing details required by the SDD template.
- **Frontend design:** stages 1–4. Use an implementation plan for stage 5 when
  the UI work spans multiple files or steps.
- **Implementation plan:** stage 5, with the approved SDD and linked source
  files as baseline context.
- **User-requested subset:** run only the requested stages and label the report
  accordingly; do not imply a full EGM review occurred.

Write each stage's complete report, including reviewer status and findings, to
the review directory. For consolidated reviews, also write an index that maps
findings to their reviewer and stage. Deduplicate overlapping findings without
hiding dissent or changing severity.

## 4. Resolve findings as Elephant

Classify every finding without changing its severity:

- **CONTEST:** the finding is inaccurate; cite checkable evidence in a file or
  decision the reviewer could read.
- **ESCALATE:** rebuttal depends on knowledge absent from all supplied files;
  ask the user instead of inventing context.
- **PROPOSE FIX:** the finding is valid; specify a precise document edit.
- **DECLINE:** record a nit or accepted trade-off that is not worth changing.

Do not contest a comprehension gap merely because the Elephant understands the
document. Do not add context solely to satisfy a reviewer when a human reader
does not need it.

Write the classifications and proposed edits to `goldfish-review-report.md` in
the document's review directory. Ask the user to accept, reject, or discuss the
proposed changes. Apply only approved document edits. If a stage is re-audited
after edits, use fresh reviewers, recalculate line ranges, and limit it to one
re-audit; remaining findings need an explicit user decision.

Only after resolution should the Elephant ask the user to approve the design.
The user may accept documented risks; the agent must not hide unresolved
findings or convert reviewer silence into approval.

## 5. Implement and check drift

Implement only from approved design artifacts. If implementation exposes a
material design gap, pause and update the approved design through the same
human-decision process before continuing.

When asked for a drift report, compare each design with the current source it
references. Report only specific differences: missing behavior, changed
contracts, different dependencies or architecture, absent files that were
described as existing, and undocumented behavior that changes the design's
contracts. Read current files rather than relying on memory. Do not update code
or design automatically; the user decides which side should change.

EGM review never substitutes for the repository's build, test, lint, security,
or code-review requirements.
