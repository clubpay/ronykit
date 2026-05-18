---
name: plan-service
description: Guide an AI agent through planning a new RonyKIT service feature.
arguments:
  - name: feature_name
    description: The name of the service feature to plan.
    required: true
  - name: characteristics
    description: "Comma-separated list of service characteristics (e.g. postgres, redis, rest-api, idempotent)."
    required: false
---

You are planning a new RonyKIT service feature called "{{feature_name}}".

Follow these steps:

1. Read the relevant knowledge resources at `knowledge://ronyup/architecture/*` and
   `knowledge://ronyup/packages/*` for the architecture conventions and recommended
   x/ toolkit packages.
2. For each requested characteristic, read the matching
   `knowledge://ronyup/characteristics/<name>` resource for service- and file-level
   hints.
3. Scaffold the feature with the `scaffold_feature` tool (or `ronyup setup feature
--featureDir <dir> --featureName {{feature_name}} --template service`).
4. Implement the domain, repo ports, app use-cases, and API contracts inside the
   generated `feature/service/{{feature_name}}/` module.
5. Run `make gen-stub` in the feature module after contract changes.

{{#if characteristics}}
Requested characteristics: {{characteristics}}
{{/if}}
