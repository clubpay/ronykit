# Ubiquitous Language

The single most important practice in Domain-Driven Design. A ubiquitous language is the shared vocabulary between developers and domain experts that is used everywhere -- in conversation, documentation, code, tests, and diagrams. It is not a glossary appended to a wiki. It is the living, evolving language that shapes how the system is built and how the team thinks about the domain.

## Why Language Matters More Than Code

Software development is fundamentally a communication problem. The hardest bugs are not off-by-one errors or null pointer exceptions -- they are misunderstandings between people. When a developer hears "account" and thinks "user login record" while the domain expert means "financial ledger position," the resulting code will be structurally wrong in ways that no amount of testing can catch.

A ubiquitous language eliminates this class of failure by establishing a single, precise vocabulary that both sides use without translation. When the language is embedded in code, every type name, method name, contract route, table name, and test name becomes a checkpoint: if a domain expert cannot read the code and recognize the concepts, the model is wrong.

### The Cost of Translation

Without a ubiquitous language, every conversation requires mental translation:

| Developer Says | Domain Expert Hears | Actual Meaning | Risk |
|---------------|--------------------|-|------|
| `UserEntity` | "User? We don't have users, we have policyholders" | The system models a concept the business does not recognize | Code does not reflect reality; edge cases are missed |
| `ProcessData()` | "Process what data? Which business operation?" | A generic method hiding domain-specific logic | Business rules buried in implementation; impossible to validate |
| `status = 3` | "What does 3 mean?" | An opaque encoding of a domain concept | Magic numbers replace meaningful domain states |
| `ServiceManager` | "Manager of what?" | A catch-all type with no domain analog | God object accumulates unrelated responsibilities |

### The Payoff of Alignment

When the language is shared:

- **Domain experts can read tests.** A test named `TestPolicy_Underwrite_RejectsApplicationAboveAuthorityLimit` that calls `policy.Underwrite(application)` is immediately meaningful. A test that calls `svc.Process(dto)` is opaque.
- **Developers catch domain errors.** When a developer writes `order.Cancel()` and the domain expert says "we don't cancel orders, we void them," the naming mismatch reveals a modeling error.
- **New team members onboard faster.** The codebase teaches the domain because the names are the domain.
- **Refactoring is safer.** Renaming `Process()` to `AdjudicateClaim()` is not just cosmetic -- it encodes domain knowledge that prevents future misuse.

## Building the Language

### Start with Domain Expert Conversations

The language does not come from developers reading documentation. It comes from intensive, iterative conversations between developers and domain experts. These conversations follow a pattern:

1. **Listen for the nouns and verbs the expert uses naturally.** "When a claim comes in, the adjuster reviews it, and then we either approve or deny the claim." The nouns are `Claim`, `Adjuster`. The verbs are `review`, `approve`, `deny`.

2. **Challenge ambiguity.** "You said 'review' -- what exactly happens during a review? Is it the same as 'assess'?" Often the expert will distinguish between terms that outsiders conflate.

3. **Propose the model.** "So a `Claim` goes through an `Adjudication` process where an `Adjuster` either `approves` or `denies` it?" The expert corrects: "Not exactly -- the adjuster makes a recommendation, but only a senior adjuster can approve claims over $10,000."

4. **Refine until the model is precise.** The conversation reveals business rules that no requirements document captured: approval authority limits, recommendation vs. decision, escalation paths.

### Model Exploration Whirlpool

Eric Evans describes a "whirlpool" process for modeling:

1. **Scenario walkthrough.** Walk through concrete business scenarios with domain experts. "A customer calls to report damage to their vehicle. What happens next?"

2. **Concept extraction.** Identify the key domain concepts that emerge: `Claim`, `Incident`, `CoverageVerification`, `DamageAssessment`.

3. **Name negotiation.** Debate and agree on names. "Should we call it a 'damage report' or a 'claim'? When does a report become a claim?"

4. **Code spike.** Quickly implement the emerging model in code to test whether it holds up under real logic.

5. **Feedback loop.** Show the code (or at least the type and method names) back to the domain expert. "Does this look right to you?"

This cycle repeats continuously throughout the project, not just at the beginning.

## Language in Code

### Naming Types After Domain Concepts

Every type in `internal/domain` should be named after a concept the business recognizes:

**Good names (domain language):**
- `LoanApplication` -- the business knows what this is
- `CreditDecision` -- an explicit outcome of an underwriting process
- `PaymentSchedule` -- a concrete domain concept
- `PolicyRenewalNotice` -- named exactly as the business document is named

**Bad names (technical language):**
- `ApplicationDTO` -- DTO is a technical pattern, not a domain concept
- `PaymentService` -- "service" is a technical role, not a domain concept; what does this service do?
- `DataProcessor` -- meaningless in domain terms
- `BaseEntityAbstractFactory` -- pure technical jargon
- Go naming smells: types ending in `Manager`, `Helper`, `Util`, `Processor`; packages named `common`, `utils`, `helpers`, `models`

RonyKit's structural names (`App` in `internal/app`, `Service` in `api/`, package `repo`) are framework scaffolding, not domain concepts, and are fine. What they contain is not: the methods on `*App`, the handler and route names, and every type in `internal/domain` must use the domain language.

### Naming Methods After Domain Operations

Methods should read like sentences a domain expert would say (`claim.SubmitForAdjudication(now)`):

**Good:**
```go
func (p *Policy) Renew(effective time.Time) error
func (c *Claim) SubmitForAdjudication(at time.Time) error
func (a *Account) ApplyInterest(rate InterestRate, period Period) error
func (o *Order) FulfillWith(s Shipment) error
```

**Bad:**
```go
func (p *Policy) Update(data map[string]any) error
func (c *Claim) Process() error
func (a *Account) Calculate() error
func (o *Order) SetStatus(status int)
```

In Go the receiver already names the concept, so the method is just the domain verb: `claim.SubmitForAdjudication`, not `claim.SubmitClaimForAdjudication`. On `*App` the receiver no longer names the concept, so the use case carries the noun with the same verb (`App.SubmitClaimForAdjudication`), and so does the contract route (`rony.UnaryName("SubmitClaimForAdjudication")`).

The difference is not cosmetic. `claim.SubmitForAdjudication(now)` tells you what the business operation is. `claim.Process()` tells you nothing -- you must read the implementation to understand what it does, which means the domain knowledge is hidden.

### Naming Events After Domain Facts

Domain events should be named as past-tense facts that a domain expert would recognize as significant business occurrences:

**Good:** `PolicyIssued`, `ClaimDenied`, `PaymentOverdue`, `MembershipExpired`

**Bad:** `PolicyUpdated`, `ClaimProcessed`, `DataChanged`, `RecordModified`

"Updated," "processed," and "changed" are technical descriptions of what happened to data. "Issued," "denied," "overdue," and "expired" are domain descriptions of what happened in the business.

## Glossary Maintenance

### The Living Glossary

Maintain a glossary document that evolves with the model. This is not a static artifact created at the start of the project and forgotten. It is a living document updated every time the language changes.

In a RonyKit workspace the glossary is §1.3 of the feature's SRS (`docs/design/<feature>-srs.md`). Each term then flows, unchanged, into the SDD domain design (§4: types and domain errors), and from there into Go identifiers in `internal/domain`, `*App` method names, contract route names, SQL table and column names, and test names. If a term is renamed in one place, it is renamed in all of them.

**What goes in the glossary:**

| Term | Definition | Context | Example |
|------|-----------|---------|---------|
| Claim | A formal request for payment under an insurance policy following a covered event | Claims processing | "The policyholder filed a claim for water damage" |
| Adjudication | The process of evaluating a claim to determine whether it is covered and how much to pay | Claims processing | "The claim is in adjudication pending the damage assessment" |
| Coverage | The set of perils and limits defined in a policy | Underwriting | "This policy provides coverage for fire but not flood" |
| Premium | The amount the policyholder pays for coverage | Billing | "The annual premium is $1,200, payable monthly" |

**What does NOT go in the glossary:**
- Technical terms (`Repository`, `Service`, `Controller`)
- Implementation details (PostgreSQL table name, API endpoint) -- these are derived from glossary terms, not entries of their own
- Generic programming concepts (`interface`, `abstract class`)

### When Terms Conflict

The same word often means different things in different parts of the business. This is expected and healthy:

- **"Account"** in billing means a financial ledger position. In authentication, it means a user login. In sales, it means a company relationship.
- **"Product"** in the catalog means something you can browse. In inventory, it means a physical item with a location. In marketing, it means a brand offering.

The solution is not to force one definition. The solution is bounded contexts: each context has its own definition, and translations happen at the boundary.

## Language Evolution

### When to Change the Language

The ubiquitous language is not fixed. It evolves as the team's understanding of the domain deepens. Signals that the language needs to change:

- **Awkward conversations.** "Well, it's kind of like a customer but not exactly -- it's more of an applicant who might become a customer." This means `Customer` is the wrong term; `Applicant` is a distinct concept.
- **Workarounds in code.** A kind field that switches behavior (e.g., `if c.Kind() == CustomerKindProspect`) often means a single type is trying to represent two different domain concepts.
- **Expert correction.** "We don't really call it a 'request' -- we call it a 'submission.'" Change the code immediately.
- **New domain insight.** "Actually, a cancellation and a voiding are different things. Cancellation is prospective; voiding is retroactive." Split the concept.

### The Refactoring Trigger

When the language changes, the code must change. This is not optional. If the team agrees that "submission" is the correct term instead of "request," then:

1. Rename the domain type: `domain.Request` becomes `domain.Submission` (and its errors: `ErrRequestInvalid` becomes `ErrSubmissionInvalid`)
2. Rename the port and adapter: `repo.RequestRepository` becomes `repo.SubmissionRepository`, in `port.go` and `internal/repo/v0`
3. Rename the events: `RequestCreated` becomes `SubmissionFiled`
4. Update the database: table `requests` becomes `submissions` (a new numbered migration, updated sqlc queries, `make sqlc`)
5. Update the API: route `/requests` becomes `/submissions` and the `UnaryName` changes (keep the old route during a deprecation window), then `make gen-stub` so consumers get the new names
6. Update the SRS glossary, the SDD, and the test names

This may seem expensive, but the cost of maintaining a divergence between language and code is far higher. Every time a developer reads `Request` and has to mentally translate to "submission," they lose context and risk introducing errors.

## How Naming Shapes Design

Names are not labels applied after the fact. They are design decisions that constrain and guide the system's evolution.

### Names Reveal Missing Concepts

When you struggle to name something, the model is telling you something is wrong:

- **"OrderProcessorHelper"** -- If you need a helper, the type it is helping probably has the wrong boundaries. The behavior likely belongs inside `Order` or in a separate, well-named domain concept.
- **"MiscService"** -- If you cannot name it, you do not understand it. Break it apart until each piece has a clear domain name.
- **"DataValidator"** -- Validation of what? By what rules? `CreditApplicationValidator` or `AddressVerifier` tells you exactly what domain rules are being enforced.

### Names Prevent Misuse

A method named `account.Debit(amount)` tells future developers exactly what this does and constrains its usage. A method named `account.Update(amount)` invites misuse because "update" could mean anything.

### Names Create Boundaries

When two concepts share a name but have different behaviors, splitting the name splits the model:

- `Customer` in sales vs. `Customer` in billing? These are different bounded contexts. The split in naming reveals the split in models.
- `Order` before payment vs. `Order` after payment? Perhaps these are `PendingOrder` and `ConfirmedOrder` -- two distinct states that deserve distinct types, not a `status` flag.

## Anti-Patterns to Avoid

### The Jargon Trap

Developers invent internal jargon that has no domain analog: "We call it a 'widget' internally." If domain experts do not use the word "widget," it does not belong in the domain model.

### The Abbreviation Trap

`CustAcctMgr` is not a ubiquitous language term. Write `CustomerAccountManager` -- or better yet, ask what the business actually calls this role. Maybe it is a `RelationshipOfficer` or an `AccountExecutive`.

### The Thesaurus Trap

Using synonyms interchangeably (`customer` / `client` / `user` / `patron`) destroys precision. Pick one term per concept and enforce it everywhere. If the business uses "client" in legal contexts and "customer" in sales contexts, those are different bounded contexts with different terms -- and both are correct within their context.

### The Persistence Trap

Naming domain concepts after their storage mechanism: `CustomerRecord`, `OrderRow`, `PaymentTable`. Domain objects are not records or rows. They are domain concepts that happen to be persisted. Name them for what they are in the domain: `Customer`, `Order`, `Payment`. The same applies to sqlc's generated row structs in `data/db`: they are persistence types, mapped to domain types inside `internal/repo/v0`, and never passed to `internal/app`.
