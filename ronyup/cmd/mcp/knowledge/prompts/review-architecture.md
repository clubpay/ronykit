---
name: review-architecture
description: Review the architecture of an existing RonyKIT service feature for best-practice compliance.
arguments:
  - name: feature_path
    description: Path to the feature directory to review.
    required: true
---
Review the RonyKIT service feature at "{{feature_path}}" for architecture best-practice compliance.

Check the following:
1. API handlers are thin — validation only, business logic delegated to internal/app.
2. Persistence is abstracted behind internal/repo/port.go interfaces.
3. Dependencies are wired through x/di.RegisterService in module.go.
4. Configuration uses x/settings with typed struct and `settings` tags.
5. Logging uses x/telemetry/logkit exclusively (no raw zap/slog/log).
6. Integration tests use x/testkit with Gnomock containers.
7. API docs are generated with x/apidoc.
8. Client stubs are up to date (make gen-stub).

Report any violations and suggest specific fixes.
