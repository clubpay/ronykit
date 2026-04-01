---
name: write-service-code
description: Guide an AI agent through writing RonyKIT service code following framework conventions.
arguments:
  - name: service_name
    description: The name of the service module (without the "mod" suffix, e.g. "auth", "ledger").
    required: true
  - name: description
    description: A brief description of what the service does.
    required: true
  - name: characteristics
    description: "Comma-separated traits (e.g. postgres, redis, cache, workflow, i18n)."
    required: false
---
You are writing service code for a RonyKIT module called "{{service_name}}mod".

Service description: {{description}}

{{#if characteristics}}
Requested characteristics: {{characteristics}}
{{/if}}

Follow the RonyKIT service structure conventions below.

## Package Layout

```
{{service_name}}mod/
  service.go          — Service struct, App(), Module(), init() with di.RegisterService
  module.go           — fx.Module wiring: repo init, datasource params, fx.Provide(settings, app, api)
  migration.go        — //go:embed internal/repo/v0/data/db/migrations (if using postgres)
  api/
    service.go        — RContext type alias, ServiceParams (fx.In), Service struct, Desc()
    api_<domain>.go   — Per-domain handler files with Input/Output DTOs + thin handler methods
  internal/
    app/app.go        — App struct (NewAppParams with fx.In), business logic methods
    domain/
      doc.go          — Package declaration
      errors.go       — Domain errors using errs.GenWrap / errs.B patterns
      <types>.go      — Domain types (plain structs, no json tags)
    repo/
      port.go         — Repository interfaces per domain concept, using domain types
      v0/adapter.go   — var Init = fx.Options(...) binding concrete repos to interfaces
      v0/<impl>.go    — Concrete repository implementations
    settings/
      settings.go     — Typed config struct with `settings` tags
      const.go        — ModuleName, ModuleVersion, RedisPrefix
```

## Key Rules

1. **Thin handlers**: API handlers only validate input, call `app` methods, and map results to output DTOs.
2. **Business logic in app**: All use-case behavior lives in `internal/app`.
3. **Repo ports**: Persistence is abstracted behind interfaces in `internal/repo/port.go` using domain types.
4. **DI wiring**: Use `x/di.RegisterService` in `service.go` init(). Wire DB/Redis params via `di.ProvideDBParams` and `di.ProvideRedisParams`.
5. **Settings**: Use `x/settings` with typed struct and `settings` tags for configuration.
6. **Logging**: Use `x/telemetry/logkit` exclusively — never raw log/slog/zap.
7. **Error handling**: Use `rony/errs` exclusively. Define domain errors with `errs.GenWrap(code, "ERROR_CODE")` or `errs.B().Code(code).Msg("ERROR_CODE").Err()`. Error codes are SCREAMING_SNAKE_CASE.

## Handler Signature Pattern

```go
type RContext = rony.UnaryCtx[*State, Action]

func (svc Service) HandlerName(ctx *RContext, in InputDTO) (*OutputDTO, error) {
    result, err := svc.app.DoSomething(ctx.Context(), in.Field)
    if err != nil {
        return nil, err
    }
    return toOutputDTO(result), nil
}
```

## Service Setup (in api/service.go Desc())

```go
func (svc Service) Desc() *desc.Service {
    return rony.Setup[*State, Action](
        "{{service_name}}",
        rony.ToInitiateState[*State, Action](&State{}),
        rony.WithUnary(svc.HandlerName, rony.POST("/v1/{{service_name}}/action")),
    )
}
```

If the service has no shared state, use `rony.EMPTY` / `rony.NOP`:

```go
type RContext = rony.SUnaryCtx

rony.Setup[rony.EMPTY, rony.NOP](
    "{{service_name}}",
    rony.EmptyState(),
    rony.WithUnary(svc.HandlerName, rony.GET("/v1/{{service_name}}/items")),
)
```

## Workflow of Implementation

1. Start with `internal/domain/` — define domain types and errors.
2. Define repository interfaces in `internal/repo/port.go`.
3. Implement business logic in `internal/app/app.go`.
4. Create API handlers in `api/` with Input/Output DTOs.
5. Wire everything in `module.go` and `service.go`.
6. Write integration tests using `x/testkit`.
7. Run `make gen-stub` to generate client stubs.

Use the `plan_service` tool first to get architecture hints and package recommendations, then `implement_service` to generate starter code.
