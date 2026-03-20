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
1. Call the `plan_service` tool with the feature name and any requested characteristics.
2. Review the architecture hints and recommended packages in the plan output.
3. If the plan looks correct, call `implement_service` to generate starter code.
4. Follow the next_steps from the plan output to complete the implementation.
5. Run `make gen-stub` in the feature module after contract changes.

{{#if characteristics}}
Requested characteristics: {{characteristics}}
{{/if}}
