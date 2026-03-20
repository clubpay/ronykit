---
keywords:
  - di
  - dependency inject
  - wire
  - lifecycle
applies_to_files:
  - module
  - service
---
Wire all service dependencies through x/di: use di.RegisterService for lifecycle management, di.StubProvider for inter-service stubs with trace propagation.

## File-Level Hint

Register via x/di.RegisterService and provide infra params through x/di helpers.
