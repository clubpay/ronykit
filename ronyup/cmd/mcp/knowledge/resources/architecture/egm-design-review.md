# Elephant–Goldfish design review

RonyKit uses the Elephant–Goldfish Model (EGM) to stress-test design artifacts
before implementation. The primary agent (Elephant) retains project context and
drafts; fresh read-only subagents (Goldfish) review without the drafting
conversation; the user decides trade-offs, findings, and approval.

The executable protocol and reviewer prompts ship with the existing
`ronykit-framework` skill. Before brainstorming, reviewing, resolving findings,
or checking drift, read:

1. `.agents/skills/ronykit-framework/references/egm.md`
2. `.agents/skills/ronykit-framework/references/egm-reviewers.md` when
   delegating a reviewer

These files include the upstream source commit and Apache-2.0 license. No
separate EGM plugin or network access is required.

## RonyKit artifact mapping

Keep the existing RonyKit templates, paths, and approval frontmatter. EGM adds
review artifacts; it does not replace SRS/SDD or frontend design requirements.

| Work | Elephant artifact | Goldfish review |
| ---- | ----------------- | --------------- |
| Unclear feature direction | Optional `docs/design/<feature>-concept.md` | Six-lens brainstorm after the user confirms the anchored hypothesis |
| Backend requirements | `docs/design/<feature>-srs.md` | Zoom out and comprehension |
| Backend design | `docs/design/<feature>-sdd.md` | Full five-stage design and implementation-readiness review |
| Frontend design | `docs/design/<app>-frontend-design.md` | Zoom out, comprehension, design critique, intermediate zoom out |
| Multi-step implementation | `docs/plans/YYYY-MM-DD-<feature>.md` | Implementation-readiness stage with the approved SDD as context |
| Design/code consistency | `docs/design/reviews/YYYY-MM-DD-drift-report.md` | Current-source comparison requested by the user |

Write review reports under `docs/design/reviews/<document-slug>/` unless the
repository has another artifact convention. Keep long drafts and reports out of
chat; link to them and discuss only the decisions needed from the user.

## Review stages

Run stages in order and reviewers within a stage in parallel:

1. **Zoom out:** root problem, simplicity, existing-component reuse, and
   architecture altitude.
2. **Comprehension:** a fresh reader paraphrases the proposal and current
   system, then identifies missing context.
3. **Design critique:** design and structural reviewers apply the feature-spec
   rubric; a style reviewer checks objective, clear writing.
4. **Intermediate zoom out:** a new reviewer examines the current document and
   stage-1 report, focusing on additions and mitigations.
5. **Implementation readiness:** readiness and codebase reviewers check exact
   files, interfaces, migrations, failure behavior, and tests.

The detailed lens assignments, rubrics, report formats, and resolution rules
are in the skill's EGM references.

## Non-negotiable rules

- A Goldfish reviewer must be a new subagent with no drafting conversation. A
  same-context review is not EGM.
- Reviewers are read-only and inspect the target plus its referenced source.
  Give them exact paths, scope, lens, and rubric.
- Retry a failed reviewer once. Record a second failure as `not run`; never
  count it as zero findings. A report with a missing reviewer is incomplete.
- If fresh subagents are unavailable, stop the EGM review and ask whether the
  user wants human-only review. Do not simulate Goldfish in the main context.
- The Elephant classifies findings as `CONTEST`, `ESCALATE`, `PROPOSE FIX`, or
  `DECLINE`, then asks the user which edits to apply. Do not change severity or
  hide unresolved findings.
- Agents write design documents with `status: draft`. Only the user approves
  them, directly or by explicitly instructing the agent to change the status.
- EGM does not replace `scaffold_feature`, `check_frontend_design`, builds,
  tests, lint, code review, or security checks.

## Drift checks

When the user asks for a drift report, compare design documents with the current
source they reference. Report only concrete differences and read the current
files instead of relying on memory. Do not automatically change either code or
design; the user decides which should change.
