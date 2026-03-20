---
keywords:
  - idempotent
applies_to_files:
  - api
  - app
  - repo
---
Design app/repo writes to be retry-safe; handlers should remain deterministic.

## File-Level Hint

Ensure behavior is idempotent and safe under retries and duplicate deliveries.
