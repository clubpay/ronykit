# AI guide for RonyKIT

Use this guide to work in the RonyKIT repository. Keep the content short, correct, and current.

## Project summary

RonyKIT is a Go toolkit for high-performance API and edge servers. It has two main entry points:

- `rony/`: High-level framework with typed handlers and default configuration. Use this package for most services.
- `kit/`: Low-level tools for custom gateways, clusters, and protocols.

Supporting components:

- `std/`: Standard gateways and clusters.
- `flow/`: Workflow helpers.
- `stub/`: Client stub generation.
- `ronyup/`: Scaffolding CLI and development tools.
- `x/`: Additional packages, including apidoc, cache, di, i18n, and telemetry.
- `example/`: Runnable examples. Standard lint and test loops exclude this directory.

## Go workspace and modules

- `go.work` defines the Go workspace and its modules.
- Each module has a `go.mod` file.
- Update `go.work` and `go.work.sum` when you add a module.
- CI uses Go `1.25`. See `.github/workflows/go.yml`.
- `go.work` declares Go `1.25.1`.

## Common commands

Run these commands from the repository root:

```sh
make setup    # Install gotestsum and golangci-lint
make test     # Test workspace modules except example and ronyup
make lint     # Run golangci-lint --fix across modules
make vet      # Run go vet across modules
make tidy     # Run go mod tidy across modules
```

## Scaffolding with ronyup

Use `ronyup` to create workspaces and features:

```sh
go install ./ronyup
ronyup setup workspace --repoDir ./my-repo --repoModule github.com/you/myrepo
ronyup setup feature --featureDir auth --featureName auth --template service
```

The default feature path is `feature/<name>/`. Use `--featurePrefix` or `--groupByTemplate` to select a different layout.

Use `setup workspace --kind fullstack` for separate `backend/` and `frontend/` directories. The Go workspace is in `backend/`. Run `setup feature` from that directory. The `devops/`, `docs/`, and AI configuration directories stay at the repository root.

Run this CI test subset from the repository root:

```sh
go test -v -cover -covermode=atomic -coverprofile=coverage.out -count=1 \
  ./kit/... \
  ./rony/... \
  ./stub/... \
  ./std/gateways/fasthttp/...
```

## Formatting and lint rules

- `.golangci.yml` configures `gofmt`, `gofumpt`, `goimports`, and `gci`.
- `.editorconfig` specifies tabs with width 2 and LF line endings.
- Do not use `io/ioutil`. The `depguard` linter blocks this package.

## Package selection

- Use `rony/` for standard services with REST or RPC endpoints.
- Use `kit/` for custom gateways, custom clusters, or protocol changes.
- Add standard gateway and cluster modules under `std/`.
- Add experimental or optional utilities under `x/`.

## API descriptions and stubs

- Use `rony.WithUnary`, `rony.WithStream`, and `rony.WithRelay` to declare handlers.
- Use REST and RPC selectors, including `GET`, `POST`, and `RPC`.
- Use `rony.WithRelay` for dynamic HTTP and WebSocket relays.
- Use `srv.ExportDesc()` and `stub/stubgen` to generate Go and TypeScript clients.
- Use `kit/desc` for direct control of descriptors.

## Additional references

- `docs/getting-started.md`: Full tutorial.
- `docs/cookbook.md`: Service patterns.
- `docs/ronyup-guide.md`: Scaffolding CLI and MCP server.
- `docs/architecture.md`: Component relationships.
- `docs/advanced-kit.md`: Low-level `kit` usage.
- `example/`: Runnable services.
- `rony/README.MD` and `kit/README.MD`: Package quick starts.

## Update this guide

Keep this guide consistent with these files:

- `go.work`: Go version and module list.
- `.github/workflows/go.yml`: CI Go version and test scope.
- `Makefile`: Available targets.
- `docs/`: Documentation structure.
