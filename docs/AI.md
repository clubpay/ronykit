# AI Guide for RonyKIT

This document gives AI assistants (and new contributors) enough context to work
effectively in this repo without guessing. Keep it short, factual, and updated.

## Project summary

RonyKIT is a Go toolkit for building high-performance API and edge servers.
There are two main entry points:

- `rony/`: batteries-included framework with typed handlers and opinionated
  defaults. Use this for most services.
- `kit/`: low-level toolkit for custom gateways, clusters, or protocols.

Supporting components include:

- `std/`: standard gateways and clusters (fasthttp, fastws, silverhttp, redis,
  p2p).
- `flow/`: workflow helpers.
- `stub/`: stub generation utilities.
- `ronyup/`: CLI for scaffolding and tooling.
- `x/`: additional packages (apidoc, cache, di, i18n, telemetry, etc).
- `example/`: runnable examples (not part of normal lint/test loops).

## Go workspace and modules

- The repo is a Go workspace (`go.work`) that includes many modules under
  `example/`, `flow/`, `kit/`, `rony/`, `ronyup/`, `std/`, `stub/`, `testenv/`,
  and `x/`.
- Each module has its own `go.mod`. When adding a new module, update `go.work`
  and `go.work.sum`.
- CI currently runs with Go `1.25` (see `.github/workflows/go.yml`).
- `go.work` currently declares `go 1.25.1`.

## Common commands

From repo root:

```sh
make setup    # installs gotestsum and golangci-lint
make test     # runs gotestsum across workspace modules (excludes example, ronyup)
make lint     # golangci-lint --fix across modules
make vet      # go vet across modules
make tidy     # go mod tidy across modules
```

## Scaffolding (ronyup)

`ronyup` scaffolds workspaces and features.

```sh
go install ./ronyup
ronyup setup workspace --repoDir ./my-repo --repoModule github.com/you/myrepo
ronyup setup feature --featureDir services/auth --featureName auth --template service
```

CI test scope (subset):

```sh
go test -v -cover -covermode=atomic -coverprofile=coverage.out -count=1 \
  ./kit/... \
  ./rony/... \
  ./stub/... \
  ./std/gateways/fasthttp/...
```

## Conventions and linting

- Formatting/linting: `gofmt`, `gofumpt`, `goimports`, `gci` via
  `.golangci.yml`.
- `.editorconfig` uses tabs with size 2 and LF.
- Avoid `io/ioutil` (blocked by `depguard` in `.golangci.yml`).

## How to choose where to implement

- Prefer `rony/` for standard services with REST/RPC endpoints.
- Use `kit/` for advanced control (custom gateway/cluster, protocol changes).
- Add new gateway/cluster bundles under `std/`.
- Put experimental or optional utilities under `x/`.

## API description and stubs

- `rony` supports declarative routes using `rony.WithUnary/WithStream` and
  REST/RPC selectors (`GET`, `POST`, `RPC`, etc).
- Use `srv.ExportDesc()` and `stub/stubgen` to generate Go/TS clients.
- For raw descriptor control, use `kit/desc`.

## Where to look for examples

- `GETTING_STARTED.MD` for a full walkthrough.
- `example/` for runnable sample services.
- `rony/README.MD` and `kit/README.MD` for quick starts.

## Updating this file

Keep this guide aligned with:

- `go.work` (Go version and module list)
- `.github/workflows/go.yml` (CI Go version and test scope)
- `Makefile` (available targets)
