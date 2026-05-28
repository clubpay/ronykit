---
name: generate-stubs
description: Guide an AI agent through generating and consuming typed client stubs for RonyKIT services.
arguments:
  - name: service_name
    description: The service module name (without "mod" suffix) to generate stubs for.
    required: true
  - name: languages
    description: "Comma-separated target languages (go, typescript). Defaults to go."
    required: false
  - name: consumer_service
    description: The name of the service that will consume the generated stubs (for cross-service wiring guidance).
    required: false
---

You are generating typed client stubs for the "{{service_name}}mod" RonyKIT service.

{{#if languages}}
Target languages: {{languages}}
{{/if}}

{{#if consumer_service}}
Consumer service: {{consumer_service}}mod
{{/if}}

## Overview

RonyKIT generates typed client stubs from service contract descriptors. Stubs provide a compile-time-safe HTTP/WebSocket client that mirrors the service's API surface. Both Go and TypeScript stubs are supported.

## Stub Generator Setup

Each service has a `gen/stub/gen.go` (or `.gotmpl`) file: a `main` package using
cobra that calls `rony.GenerateStub`. The type parameters are inferred from the
`api.Service{}.Desc()` argument, so callers do not pass them explicitly:

```go
package main

import (
    "github.com/spf13/cobra"

    "github.com/clubpay/ronykit/rony"
    "github.com/clubpay/ronykit/stub/stubgen"

    "your-module/feature/{{service_name}}/api"
)

var Flags = struct {
    PackageName string
    DstDir      string
}{}

var RootCmd = &cobra.Command{Use: "gen"}

var GenGoCmd = &cobra.Command{
    Use: "go",
    RunE: func(_ *cobra.Command, _ []string) error {
        return rony.GenerateStub(
            Flags.PackageName,
            "",
            Flags.DstDir,
            stubgen.NewGolangEngine(stubgen.GolangConfig{
                PkgName: Flags.PackageName,
            }),
            api.Service{}.Desc(),
        )
    },
}

var GenTypescriptCmd = &cobra.Command{
    Use: "ts",
    RunE: func(_ *cobra.Command, _ []string) error {
        return rony.GenerateStub(
            Flags.PackageName,
            "",
            Flags.DstDir,
            stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{
                GenerateSWR: true,
            }),
            api.Service{}.Desc(),
        )
    },
}

func init() {
    RootCmd.PersistentFlags().StringVarP(&Flags.PackageName, "pkg-name", "n", "{{service_name}}stub", "package name")
    RootCmd.PersistentFlags().StringVarP(&Flags.DstDir, "output-dir", "o", "../stub", "output directory")
}

func main() {
    RootCmd.AddCommand(GenGoCmd, GenTypescriptCmd)
    if err := RootCmd.Execute(); err != nil {
        panic(err)
    }
}
```

## Makefile Targets

The scaffold's Makefile exposes:

```makefile
gen-go-stub:
	@go run ./gen/stub/gen.go go -o ./stub/{{service_name}}stub
	@go mod tidy
	@go fmt ./...

gen-ts-stub:
	@go run ./gen/stub/gen.go ts -o ./stub/{{service_name}}stub-typescript
	@go mod tidy
	@go fmt ./...

gen-stub: gen-go-stub
```

`gen-stub` only runs the Go target by default; extend it to depend on
`gen-ts-stub` as well when the service ships a TypeScript client.

## Generated Output

| Language   | Output Path                             | Contents                                      |
| ---------- | --------------------------------------- | --------------------------------------------- |
| Go         | `stub/{{service_name}}stub/stub.go`     | Typed client struct with methods per contract |
| TypeScript | `stub/{{service_name}}stub-typescript/` | Typed fetch client + SWR hooks (if enabled)   |

## Consuming Go Stubs (Cross-Service)

In the consuming service's `module.go`, wire the stub via `x/di.StubProvider`:

```go
var appModule = fx.Module(
    settings.ModuleName,
    v0repo.Init,
    fx.Provide(settings.New, app.New, api.New),
    di.ProvideDBParams[settings.Settings](MigrationFS),
    di.ProvideRedisParams[settings.Settings](),
    datasource.InitDB("", ""),
    datasource.InitRedis("", ""),
    di.StubProvider[settings.Settings, I{{service_name}}Stub, *{{service_name}}stub.Stub](
        settings.ModuleName,
        "{{service_name}}HostPort",
        {{service_name}}stub.New,
    ),
)
```

Expose the corresponding host/port in the consuming service's settings:

```go
type Settings struct {
    Services struct {
        {{service_name}}HostPort string `settings:"{{service_name}}_host_port"`
    } `settings:"services"`
}
```

Then inject the stub interface into the app layer via fx:

```go
type NewAppParams struct {
    fx.In
    {{service_name}}Stub I{{service_name}}Stub
    // ... other deps
}
```

## Consuming TypeScript Stubs

The TypeScript stubs generate:

- A typed client class with methods per endpoint
- SWR hooks for React data fetching (when `GenerateSWR: true`)

Import and use:

```typescript
import { {{service_name}}Client } from './{{service_name}}stub-typescript';

const client = new {{service_name}}Client({ baseURL: 'https://api.example.com' });
const result = await client.getItem({ id: 123 });
```

## Using the Stub Client Directly

For ad-hoc or testing scenarios, you can use the `stub` package directly:

```go
import "github.com/clubpay/ronykit/stub"

s := stub.New("api.example.com", stub.Secure())
ctx := s.REST().
    GET("/v1/{{service_name}}/items/123").
    SetHeader("Authorization", "Bearer "+token).
    DefaultResponseHandler(func(ctx context.Context, r stub.RESTResponse) *stub.Error {
        var out ItemResponse
        err := r.Decode(&out)
        return nil
    }).
    Run(ctx)
defer ctx.Release()
```

## When to Regenerate

Regenerate stubs whenever you:

- Add, remove, or rename a contract
- Change input/output DTO types
- Modify route paths or HTTP methods
- Add or remove route selectors

Run: `make gen-stub` (and `make gen-ts-stub` if the service ships TypeScript).

## Checklist

1. Ensure `api.Service{}.Desc()` returns all contracts correctly.
2. Create or update `gen/stub/gen.go` with both Go and TypeScript subcommands.
3. Run `make gen-stub` (and optionally `make gen-ts-stub`) to generate stubs.
4. Verify generated stubs compile: `cd stub/{{service_name}}stub && go build ./...`.
5. In consuming services, wire stubs via `di.StubProvider` in `module.go`.
6. Update consuming service settings with the stub host/port config.
7. Always regenerate after contract changes.
