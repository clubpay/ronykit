# Junie Project Guidelines — RonyKIT

These guidelines tell Junie how to work effectively in this repository.

Project overview
- RonyKIT is a collection of Go modules that together provide an extendable and high‑performance toolkit for building API/Edge servers.
- The framework is split into multiple modules and standard implementations, and organized as a Go workspace.

Repository layout (high level)
- Core modules: kit, rony, flow, stub, boxship, ronyup, contrib, util
- Standard implementations:
  - Gateways: std/gateways/fasthttp, std/gateways/fastws, std/gateways/silverhttp
  - Clusters: std/clusters/p2pcluster, std/clusters/rediscluster
- Examples: example/ex-01-rpc, ex-02-rest, ex-03-cluster, ex-04-stubgen, ex-05-counter, ex-06-counter-stream, ex-08-echo, ex-09-mw
- Docs and assets: docs/

Daily workflow for Junie
1) Install tools (once per environment)
   - make setup
   - This installs gotestsum which the test script relies on.

2) Run tests (required before submitting changes)
   - Preferred: make test
   - What it does: runs scripts/run-test.sh, which iterates over key modules and examples, executes go test with coverage via gotestsum, and summarizes coverage with go tool cover.
   - Note: Because this is a multi-module workspace, running go test ./... at the repo root will not mirror the curated list used by the project; use make test.

3) Build/compile
   - There is no single top-level build target; build individual modules or examples with standard Go commands:
     - Example: cd example/ex-01-rpc && go build ./...
   - Some modules contain tools/CLIs (e.g., ronyup, boxship) that can be built similarly.

4) Formatting and basic checks
   - Run go fmt ./... and go vet ./... in the module you modified (or at repo root for a quick pass) before submitting.
   - Keep imports organized and run go mod tidy in any module where dependencies changed.

5) Submitting changes checklist
   - Keep edits minimal and targeted to the issue at hand.
   - Update or add documentation when behavior changes (README.MD or relevant package README).
   - Run make test and ensure all tests in the curated list pass.
   - Ensure the repository still builds for affected modules.

Makefile targets
- setup: installs required tools (gotestsum)
- test: runs the project’s curated test suite via scripts/run-test.sh
- cleanup: executes scripts/cleanup.sh for housekeeping
- new-version-patch / new-version-minor: internal version bump helpers

Useful references
- Project README: README.MD (overview, structure, and basic commands)
- Contribution guide: CONTRIBUTING.md
- Licenses and policies: LICENSE, CODE_OF_CONDUCT.md, COMPLIANCE.md

Notes for non-standard behavior
- Tests: The repository uses a curated list of packages and examples for testing (see scripts/run-test.sh). Always prefer make test over ad‑hoc go test ./... at the workspace root.
- Coverage: scripts/run-test.sh generates coverage.out per package and summarizes with go tool cover.
