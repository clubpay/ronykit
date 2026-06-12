# SRS template (IEEE 830 style)

Use this outline when writing `docs/design/<feature_name>-srs.md`. Replace bracketed placeholders with concrete content. Keep requirements testable and numbered.

## Required frontmatter (approval gate)

The file **must** begin with this YAML frontmatter. The `scaffold_feature` tool refuses to run until this document has `status: approved`:

```yaml
---
feature: <feature_name>
document: SRS
status: draft   # set to "approved" ONLY after the user reviews and approves
---
```

Write `status: draft` when you create the document. **Never** set `status: approved` yourself — only the user approves it (by editing the file or explicitly telling you to). This is enforced by `scaffold_feature`, not just convention.

## 1. Introduction

### 1.1 Purpose

[Why this feature exists; who consumes it.]

### 1.2 Scope

[What is in scope and explicitly out of scope.]

### 1.3 Definitions, acronyms, abbreviations

| Term | Definition |
|------|------------|
|      |            |

### 1.4 References

- [Related SRS, ADRs, external APIs, standards]

### 1.5 Overview

[One-paragraph summary of the rest of the document.]

## 2. Overall description

### 2.1 Product perspective

[How this feature fits the workspace; upstream/downstream systems.]

### 2.2 Product functions

[High-level capabilities in plain language.]

### 2.3 User classes and characteristics

[Operators, other services, end users, admin roles.]

### 2.4 Operating environment

[Deployment, Postgres/Redis, Temporal, etc.]

### 2.5 Design and implementation constraints

[RonyKIT module name, characteristics (postgres, cache, workflow), naming rules.]

### 2.6 Assumptions and dependencies

[Assumptions about data, other features, third-party services.]

## 3. Specific requirements

### 3.1 Functional requirements

Number each requirement (FR-001, FR-002, …). Each entry: **description**, **priority** (Must/Should/Could), **acceptance criteria**.

| ID     | Requirement | Priority | Acceptance criteria |
|--------|-------------|----------|---------------------|
| FR-001 |             | Must     |                     |

### 3.2 API requirements (external interface)

List each operation: name, HTTP method + path (or RPC predicate), auth, idempotency, request/response summary.

| Operation | Method / route | Auth | Summary |
|-----------|----------------|------|---------|
|           |                |      |         |

### 3.3 Data requirements

[Entities persisted; retention; migrations needed.]

### 3.4 Inter-service requirements

[Other feature modules called via generated stubs.]

### 3.5 Non-functional requirements

| ID      | Category                  | Requirement |
|---------|---------------------------|-------------|
| NFR-001 | Performance               |             |
| NFR-002 | Security                  |             |
| NFR-003 | Reliability / idempotency |             |
| NFR-004 | Observability             |             |

## 4. Appendices

[Open questions, future work, glossary extensions.]
