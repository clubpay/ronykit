---
name: generate-stubs
description: Guide an AI agent through generating and consuming typed client stubs for RonyKIT services.
arguments:
  - name: service_name
    description: The service module name (without "mod" suffix) to generate stubs for.
    required: true
  - name: languages
    description: "Comma-separated target languages (go, typescript). Defaults to both."
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

Each service has a `gen/stub/gen.go` file — a main package using cobra that calls `rony.GenerateStub`:

```go
package main

import (
    "fmt"
    "os"

    "github.com/clubpay/ronykit/rony"
    "github.com/clubpay/ronykit/stub/stubgen"
    "github.com/spf13/cobra"

    "your-module/feature/service/{{service_name}}mod/api"
)

func main() {
    root := &cobra.Command{Use: "gen-stub"}

    root.AddCommand(
        &cobra.Command{
            Use: "go",
            RunE: func(cmd *cobra.Command, args []string) error {
                return rony.GenerateStub[rony.EMPTY, rony.NOP](
                    "{{service_name}}", "{{service_name}}stub", "./stub/{{service_name}}stub",
                    stubgen.NewGolangEngine(stubgen.GolangConfig{
                        PkgName: "{{service_name}}stub",
                    }),
                    api.Service{}.Desc(),
                )
            },
        },
        &cobra.Command{
            Use: "ts",
            RunE: func(cmd *cobra.Command, args []string) error {
                return rony.GenerateStub[rony.EMPTY, rony.NOP](
                    "{{service_name}}", "", "./stub/{{service_name}}stub-typescript",
                    stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{
                        GenerateSWR: true,
                    }),
                    api.Service{}.Desc(),
                )
            },
        },
    )

    if err := root.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

If the service uses typed state, replace `rony.EMPTY, rony.NOP` with the actual state/action types and pass the matching `SetupOption` from `api.Service{}.Desc()`.

## Makefile Targets

Add these targets to the service's Makefile:

```makefile
gen-go-stub:
	go run ./gen/stub go

gen-ts-stub:
	go run ./gen/stub ts

gen-stub: gen-go-stub gen-ts-stub
```

## Generated Output

| Language | Output Path | Contents |
|----------|------------|----------|
| Go | `stub/{{service_name}}stub/stub.go` | Typed client struct with methods per contract |
| TypeScript | `stub/{{service_name}}stub-typescript/` | Typed fetch client + SWR hooks (if enabled) |

## Consuming Go Stubs (Cross-Service)

In the consuming service's `module.go`, wire the stub via `x/di`:

```go
var diDatasource = fx.Options(
    di.ProvideDBParams[settings.Settings](MigrationFS),
    di.ProvideRedisParams[settings.Settings](),
    di.Provide{{service_name}}Stub(),
)

func di{{service_name}}Stub() fx.Option {
    return di.ProvideStorageStub[settings.Settings](settings.ModuleName)
}
```

The stub host/port is read from the consuming service's settings:

```go
type Settings struct {
    ServicesConfig struct {
        {{service_name}}HostPort string `settings:"{{service_name}}_host_port"`
    } `settings:"services"`
}
```

Then inject the stub into the app layer via fx:

```go
type NewAppParams struct {
    fx.In
    {{service_name}}Stub *{{service_name}}stub.Stub
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
        // ...
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

Run: `make gen-stub`

## Checklist

1. Ensure `api.Service{}.Desc()` returns all contracts correctly.
2. Create or update `gen/stub/gen.go` with both Go and TypeScript subcommands.
3. Run `make gen-stub` to generate stubs.
4. Verify generated stubs compile: `cd stub/{{service_name}}stub && go build ./...`
5. In consuming services, wire stubs via `di.ProvideStorageStub` in `module.go`.
6. Update consuming service settings with the stub host/port config.
7. Always regenerate after contract changes.
