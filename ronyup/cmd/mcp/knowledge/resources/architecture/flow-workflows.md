Use `flow` for durable process orchestration.

Create the Temporal integration in datasource/module wiring (`flow.NewBackend` + `flow.NewSDK`), initialize typed registrations with `sdk.InitWithState(app)`, and start/stop the SDK from service lifecycle hooks. Authenticate with `flow.NewAPIKeyStaticCredentials` (or `NewAPIKeyDynamicCredentials` / `NewMTLSCredentials`); do not import the Temporal SDK to build credentials.

Keep a strict split:

- workflow files orchestrate only (activities, timers, selectors, child workflows, continue-as-new),
- activity files execute side effects (DB writes, HTTP/Kafka/stub calls) using `ctx.S()` dependencies.

Name workflows and activities with stable, namespaced identifiers (`Feature/Action`).

Build deterministic workflow IDs from business keys and expose explicit signal-name helpers for external wakeups.
