---
name: plan-service
description: Plan a new RonyKIT service feature — start with SRS and SDD before scaffolding (use design-new-service for the full workflow).
arguments:
- name: feature_name
  description: The name of the service feature to plan.
  required: true
- name: characteristics
  description: "Comma-separated list of service characteristics (e.g. postgres, redis, rest-api, idempotent)."
  required: false
---

You are planning a new RonyKIT service feature called "{{feature_name}}".

**Use the document-first workflow.** Do not scaffold or implement until SRS and SDD are written and approved.

## Recommended path

1. **SRS** — MCP prompt `write-srs` (or Phase 1 of `design-new-service`) → `docs/design/{{feature_name}}-srs.md`
2. **SDD** — MCP prompt `write-sdd` (or Phase 2 of `design-new-service`) → `docs/design/{{feature_name}}-sdd.md`
3. **Scaffold** — `scaffold_feature` tool
4. **Implement** — MCP prompt `write-service-code`, following the SDD

Read `knowledge://ronyup/architecture/design-documents` for gate rules and templates.

{{#if characteristics}} Requested characteristics: {{characteristics}} {{/if}}

## Planning checklist (after SDD is approved)

1. Read relevant `knowledge://ronyup/architecture/*` and `characteristics/*` resources.
2. Scaffold with `scaffold_feature` (or `ronyup setup feature --featureDir <dir> --featureName {{feature_name}} --template service`).
3. Implement domain, repo ports, app use-cases, and API contracts per the SDD.
4. Run `make gen-stub` in the feature module after contract changes.
