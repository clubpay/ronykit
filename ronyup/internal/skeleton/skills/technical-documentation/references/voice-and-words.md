# Voice and Words

Deepens SKILL.md §2 (Voice: You, Active, Present, Timeless) and §3 (Sentences and Words). Owns rule IDs V1-V10 and W1-W8 — cite these IDs in audit findings; S9 ("click here") and P8 (placeholders) are cited here but explained in their owner files.

## Contents
- [Second person](#second-person-v1) · [Active voice](#active-voice-v2) · [Present tense](#present-tense-v3)
- [Please, simply, and excessive claims](#please-simply-and-excessive-claims-v4-v5) · [Anthropomorphism](#anthropomorphism-v6)
- [Timeless documentation](#timeless-documentation-v7) · [Contractions \[EN\]](#contractions-en-v8)
- [Inclusive language](#inclusive-language-v9) · [Global audience](#global-audience-v10)
- [Condition before instruction](#condition-before-instruction-w1) · [Short sentences, one idea](#short-sentences-one-idea-w2)
- [Abbreviations](#abbreviations-w3-and-latin-abbreviations-en-w4) · [can/may/might \[EN\]](#canmaymight-en-w5)
- [Jargon](#jargon-w6) · [Word list \[EN\]](#word-list-en-w7)
- [American spelling and serial comma \[EN\]](#american-spelling-and-serial-comma-en-w8) · [Non-English documents](#non-english-documents)

## Second person (V1)

Address the reader as "you." Never "we" (hides who has to act) or "the user" (turns the reader into a third party watching someone else's instructions). Default to the imperative for steps — the subject "you" is implied, not written.

| Avoid | Use instead | Why |
|---|---|---|
| "We recommend restarting the Nimbus CLI daemon after a config change." | "Restart the Nimbus CLI daemon after you change the config." | "We" hides who has to act |
| "The user must set an API key before calling the endpoint." | "Set an API key before you call the endpoint." | "The user" makes the reader a bystander in their own instructions |
| "You should click Deploy to start the build." | "Click **Deploy** to start the build." | Imperative drops the throat-clearing subject |

**The one allowed "we":** Google, or the team that owns the product, speaking as the actual actor — stating a design decision, not giving an instruction. Rare, and confined to prose about the product's history or rationale.
- Allowed: "We built the Vantage API to replace polling with webhooks."
- Not allowed: "We suggest you enable webhooks." → "Enable webhooks."

Source: person

## Active voice (V2)

**Spot it:** a form of "be" (is, was, are, were, been, being) followed by a past participle, with the actor missing or trailing in a "by" phrase.

| Passive | Active | Note |
|---|---|---|
| "The manifest is validated by the Kiln build server." | "The Kiln build server validates the manifest." | Actor is named — no reason for passive |
| "Deployments are triggered when a tag is pushed." | "Pushing a tag triggers a deployment." | Actor recoverable — rewrite active |

**Three allowed passive cases:**
1. Actor unknown: "The request was rejected by an upstream proxy." → allowed as "The request was rejected" when the Wayfinder SDK genuinely can't identify which proxy.
2. Actor irrelevant to the reader's task: "Sessions are rotated every 15 minutes." — the reader needs the interval, not the internal job that does the rotating.
3. To emphasize the object over the actor: "Expired records are purged nightly." — the record is the point; naming the cron job would bury it.

Source: voice

## Present tense (V3)

Default to present tense for behavior. Reserve "will" for effects genuinely later than the action described, not for the next line of the same sequence.

| Rewrite to present | Legitimate "will" |
|---|---|
| "The server will send an ack." → "The server sends an ack." | "If you delete a project, Fleetlog will permanently remove its logs after a 90-day grace period." — the effect is deferred by a stated delay, not immediate |
| "The Fleetlog service will purge logs older than 30 days nightly." → "The Fleetlog service purges logs older than 30 days nightly." | "After three consecutive failed health checks, the load balancer will mark the instance unhealthy." — a threshold-triggered future event |

Rule of thumb: if the result follows directly from the action in the same step, use present tense. If it arrives after a stated delay or a later condition, "will" is accurate.

Source: tense

## Please, simply, and excessive claims (V4, V5)

**"Please":** omit from instructions. Reserve it for asking the reader's permission or forgiveness, not for softening a command.
- "Please click Save to persist your changes." → "Click **Save** to persist your changes."
- Allowed: "Please allow up to 24 hours for DNS changes to propagate." — asking for patience, not issuing a step.

**"Simply / easily / just / quickly":** omit. They grade the reader's experience for them; if a step really is easy, the reader notices without being told.
- "Just add the `--watch` flag to simply enable live reload." → "Add the `--watch` flag to enable live reload."

**Superlatives, absolutes, and competitor comparisons:** cut, or replace with a verifiable, specific claim.

| Avoid | Use instead |
|---|---|
| "The fastest way to deploy" | "One way to deploy" — or a stated number: "deploys in under 10 seconds on the free tier" |
| "Never loses a message" | "Retries delivery up to five times before moving the message to a dead-letter queue" |
| "Faster than Relay's queue" | Cut the comparison, or link to a published benchmark |

Source: tone, excessive-claims

## Anthropomorphism (V6)

Software doesn't want, see, think, know, tell, or complain. Name what actually happens.

| Avoid | Use instead | Example |
|---|---|---|
| wants | requires, needs | "The build script wants a `NODE_ENV` value." → "The build script requires a `NODE_ENV` value." |
| sees | detects | "The linter sees an unused import." → "The linter detects an unused import." |
| thinks | determines, evaluates | "The scheduler thinks the job is stuck." → "The scheduler determines the job is stuck after a 10-minute timeout." |
| knows | stores, has | "The cache knows the last ETag." → "The cache stores the last ETag." |
| tells | notifies, reports | "The webhook tells the queue the job failed." → "The webhook reports the failure to the queue." |
| complains | returns an error, logs | "The parser complains about invalid syntax." → "The parser returns a syntax error." |

Source: anthropomorphism

## Timeless documentation (V7)

Cut "currently," "now," "new," "soon," and "at the time of writing." A doc that never refers to today stays correct for as long as the behavior holds; one anchored to today starts rotting the day it ships.

| Avoid | Use instead |
|---|---|
| "Currently, the Cortex API rate-limits requests to 100/min." | "The Cortex API rate-limits requests to 100/min." |
| "This new dashboard shows deploy history." | "The dashboard shows deploy history." |
| "Soon you'll be able to export CSV." | Cut it — document only what's shipped |

**Never pre-announce.** A feature that isn't released yet doesn't belong in the docs, even hedged as "coming soon."

**Version differences aren't a timeless-docs violation** — they name a fact tied to a version number, not to the calendar:
- "In Nimbus CLI 2.x and later, `--dry-run` prints a diff before applying changes." — fine, because the boundary is the version, not "now."

Source: future, timeless-documentation

## Contractions [EN] (V8)

Contractions read as conversational, not sloppy, and Google's guide allows them. For negations, prefer the contraction over the two-word form.

| Prefer | Over |
|---|---|
| isn't | is not |
| don't | do not |
| won't | will not |

- "The API key is not valid for staging." → "The API key isn't valid for staging."
- "Do not delete the retention record." → "Don't delete the retention record."

Source: contractions

## Inclusive language (V9)

| Avoid | Use instead |
|---|---|
| master / slave | primary / replica, controller / worker |
| blacklist / whitelist | denylist / allowlist (or blocklist / safelist) |
| sanity check | final check |
| dummy (variable, value) | placeholder |
| crazy | unexpected, erratic |
| cripple | disable, degrade |
| guys | everyone, team, folks |
| he / she (generic) | they |

**Fix pairs together.** Replacing only "blacklist" with "denylist" while leaving "whitelist" untouched keeps the asymmetry the swap was meant to remove — change both halves of a pair in the same edit, even when only one half appears on the page in front of you.

**Gender-neutral "they":** use it for a person of unspecified gender, singular or plural, instead of switching to "he or she" or alternating pronouns.
- "A developer must configure his API key before deploying." → "A developer must configure their API key before deploying."

Source: inclusive-documentation

## Global audience (V10)

| Avoid | Use instead | Why |
|---|---|---|
| "Once you deploy the service, health checks start." | "After you deploy the service, health checks start." | "Once" reads as a time word in some dialects and a conditional in others |
| "This endpoint hits the ground running with zero config." | "This endpoint works with no configuration." | Idioms don't translate |
| "Configure it like setting up a Thanksgiving dinner — plan ahead." | Cut the analogy; describe the steps directly | Culture-bound reference |
| "the customer order fulfillment status notification service" | "the service that notifies customers about fulfillment status" | Long noun stacks don't parse for non-native readers |
| "The team, the config file having been updated, redeployed." | "The team updated the config file, then redeployed." | Keep subject-verb-object order |

Date, time, currency, and number formatting for a global audience are S11 (structure-and-formatting.md) — this section covers sentence-level and word-choice habits only.

Source: translation, global-audience

## Condition before instruction (W1)

State the condition or goal first, so the reader can tell whether the step applies to them before they act on it.

1. "Click **Delete** if you want to remove the workspace." → "To remove the workspace, click **Delete**."
2. "Restart the Anchor auth service after you edit `config.yaml`." → "After you edit `config.yaml`, restart the Anchor auth service."
3. Multi-condition: "Retry the request if the response is a 503 and a `Retry-After` header is present, and otherwise fail immediately." → "If the response is a 503 and includes a `Retry-After` header, retry the request. Otherwise, fail immediately."

Source: sentence-structure

## Short sentences, one idea (W2)

One clause, one idea. When a sentence accumulates a "which" clause, a comma-and, and a second instruction, split it into separate sentences.

Before: "The Vantage API returns a 429 status code when you exceed the rate limit, which is 100 requests per minute for free-tier accounts, and you should back off using the Retry-After header."

After: "The Vantage API returns a 429 status code when you exceed the rate limit. Free-tier accounts are limited to 100 requests per minute. Back off using the value in the `Retry-After` header."

**Convert a "which" clause into a list when it enumerates items:**

Before: "The deploy command validates the manifest, which checks the schema, the image tag, and the resource limits."

After: "The deploy command validates the manifest. It checks:
- The schema
- The image tag
- The resource limits"

Source: tech-writing/one "Short sentences"

## Abbreviations (W3) and Latin abbreviations [EN] (W4)

**First use:** spell out the term with the abbreviation in parentheses, then use the abbreviation for the rest of the page.
- "Configure the command-line interface (CLI) before running the first build. The CLI reads `~/.nimbusrc` on startup."

**Universally known exceptions:** skip the spell-out for terms every reader already knows — URL and HTML, and similarly ubiquitous ones (inferred: API, CPU).

**Latin abbreviations** don't translate and scan poorly in the middle of a sentence — spell out the meaning instead.

| Avoid | Use instead |
|---|---|
| e.g. | for example |
| i.e. | that is |
| etc. | omit, or finish the list |
| vs. | versus, or "compared with" |
| cf. (inferred) | see, or compare |

Source: abbreviations, word-list

## can/may/might [EN] (W5)

| Modal | Meaning | Example |
|---|---|---|
| can | ability | "Free-tier accounts can make up to 100 requests per minute." |
| may | permission | "You may cache a response for up to 60 seconds." |
| might | possibility | "The migration might take several minutes for databases over 10 GB." |

Don't use "may" for possibility — "The build may fail" reads as the build having permission to fail. Use "might."

Source: word-list

## Jargon (W6)

Jargon is a defect only for the reader who doesn't have it. Define, link, or replace it based on the reader named at intake.

| For this reader | Do this |
|---|---|
| Experienced backend engineers | Use the term as-is: "The write is idempotent." |
| Mixed technical audience | Define inline on first use: "The write is idempotent — repeating it produces the same result." |
| Non-technical or new readers | Link to a concept page, or replace with plain language: "Repeating the request is safe; it won't create duplicates." |

Source: jargon

## Word list [EN] (W7)

"Click here" is not a word-list row — it's rule S9 (structure-and-formatting.md), a link-text defect, not a word choice.

Entries owned by their own rule aren't repeated here — please (V4), just/simply (V5), e.g./i.e./etc. (W4), the inclusive-language pairs (V9), and once→after (V10) live in their sections; "click here" is rule S9 in structure-and-formatting.md.

| Avoid | Use instead | Why |
|---|---|---|
| above / below | preceding / following | Breaks on reflow, print, and translation |
| abort / kill | stop, cancel, end | Violent connotation |
| log in | sign in (unless the product itself says "log in") | Google's preferred term |
| setup (as a verb) | set up | "Setup" is the noun; "set up" is the verb |
| check box | checkbox | One word |
| and/or | pick one | Ambiguous |
| in order to | to | Wordy |
| desire | want | Plainer |
| leverage | use | Jargon |
| utilize | use (inferred) | Plainer |
| pop-up / dialog box | dialog | Google's preferred term |
| e-mail | email (don't use it as a verb) | Modern spelling; not a verb |
| Internet | internet | Common noun now |
| back-end / front-end | backend / frontend | One word, no hyphen |
| file name | filename | One word |
| Id / id | ID | Always capitalized |
| admin | administrator (except literal UI labels) | Plainer, except where the UI itself reads "Admin" |
| application | app (for end-user programs) | Google's preferred term in consumer contexts |
| click and drag | drag | Simpler |
| via | through, by using | Latin-derived; doesn't localize |
| allows you to | lets you / you can | Wordy |
| wish | want | Plainer |
| terminate | end, stop | Plainer |
| enable (a person) | let, lets | "Enable" a feature, not a person |
| display (intransitive) | appears | "Display" needs an object |
| execute | run | Plainer |
| illegal | invalid, not allowed | "Illegal" implies law-breaking |
| foo / bar | meaningful names in samples | Realistic names read better and copy-paste safely |
| Note that | omit | Filler opener |
| going forward | omit | Filler; also a timeless-docs violation |
| it's / its | "it's" only for "it is"; "its" is possessive | Commonly confused |
| toggle (as a verb) | turn on, turn off | Plainer |
| uncheck | clear | Google's UI term |
| unselect | deselect | Google's UI term |

Source: word-list

## American spelling and serial comma [EN] (W8)

| UK | US (use this) |
|---|---|
| colour | color |
| behaviour | behavior |
| licence (noun) | license |
| centre | center |
| optimise | optimize |

**Serial comma:** place a comma before the final "and" or "or" in a list of three or more.
- "Install the CLI, configure the API key and run the migration." → "Install the CLI, configure the API key, and run the migration."

Source: highlights, commas

## Non-English documents

The authoritative split lives in [audit-checklist.md](audit-checklist.md), "Non-English documents": the `[EN]` rules — contractions (V8), Latin abbreviations (W4), can/may/might (W5), the word list (W7), spelling and the serial comma (W8) — are skipped, and every other rule in this file, spelling out abbreviations (W3) included, applies in any language.

Never translate a document unless asked (skill rule) — edit or write in the language the source already uses, and flag translation as a separate task if one looks needed.
