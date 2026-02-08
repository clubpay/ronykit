# RonyKIT

[![codecov](https://codecov.io/gh/clubpay/ronykit/branch/main/graph/badge.svg?token=6DALKCyYXv)](https://codecov.io/gh/clubpay/ronykit)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fclubpay%2Fronykit.svg?type=shield&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fclubpay%2Fronykit?ref=badge_shield&issueType=license)
[![Go Report Card](https://goreportcard.com/badge/github.com/clubpay/ronykit)](https://goreportcard.com/report/github.com/clubpay/ronykit)

RonyKIT is a Go toolkit for building high-performance API and edge servers. It gives you two entry points: a
batteries-included framework (`rony`) and a low-level toolkit (`kit`) so you can pick the right level of control
without rewriting your handlers.

## Choose your entry point

- [rony](./rony/README.MD): recommended, fast setup, opinionated defaults.
- [kit](./kit/README.MD): low-level building blocks for custom gateways, clusters, or protocols.
- [stub](./stub/README.MD): client stubs (work in progress).

## Why RonyKIT

- Define contracts once; serve REST and RPC from the same handlers.
- Declarative service descriptors with strong typing via Go generics.
- Pluggable gateways and clusters (fasthttp, gnet, redis, p2p).
- Performance-oriented by design with minimal framework overhead.
- Tooling for scaffolding and upgrades via `ronyup`.

## Getting started

```shell
go install github.com/clubpay/ronykit/ronyup@latest
ronyup setup -d ./my-project -m github.com/you/myproject -p MyProject
```

Read the [Getting Started Guide](./GETTING_STARTED.MD) for the full walkthrough.

## Architecture

![Architect](./docs/architect.png)

## Examples

Browse runnable examples in [example](./example).

## Performance

Benchmarks can help estimate framework overhead relative to business logic. See the
[TechEmpower JSON benchmark](https://www.techempower.com/benchmarks/#hw=ph&test=json&section=data-r22&l=zijocf-cn3)
for a reference point and compare against your own workloads.

![Benchmark Results](./docs/benchmark.png)

## Project layout

- `rony`: batteries-included framework built on RonyKIT
- `kit`: core building blocks and interfaces
- `std`: standard gateway and cluster bundles
- `flow`: workflow helpers
- `stub`: stub generation utilities
- `ronyup`: project setup and upgrade tool
- `example`: runnable examples

## Build commands

```bash
# Install required tools
make setup

# Run tests
make test

# Clean up the project
make cleanup

# Update version (patch)
make new-version-patch

# Update version (minor)
make new-version-minor
```
