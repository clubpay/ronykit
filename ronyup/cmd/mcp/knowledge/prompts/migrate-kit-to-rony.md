---
name: migrate-kit-to-rony
description: Plan migration of a direct kit-based project to the rony module incrementally.
arguments:
  - name: project_scope
    description: Services/modules to migrate first.
    required: true
  - name: constraints
    description: "Non-functional constraints such as zero-downtime, strict backward compatibility, or deadline."
    required: false
---
Create a migration guide and execution plan to move this project from direct `kit` usage to the `rony` module.

Project scope: {{project_scope}}

Follow these steps:
1. Read architecture resource `knowledge://ronyup/architecture/migrating-kit-to-rony`.
2. Inventory current `kit` usage and group by transport, app, repo, and infra concerns.
3. Produce a phased migration plan with explicit rollback and verification checkpoints.
4. Define contract-compatibility checks (routes, payloads, error codes) before cutover.
5. Specify the first service/module to migrate and list concrete file-level tasks.
6. Include required validation steps (`go test ./...`, integration tests, `make gen-stub` when contracts change).

{{#if constraints}}
Constraints: {{constraints}}
{{/if}}
