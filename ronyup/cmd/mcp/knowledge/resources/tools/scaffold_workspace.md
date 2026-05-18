---
name: scaffold_workspace
---

Initialize a new RonyKIT workspace at the given directory by delegating to `ronyup setup workspace`.

## Extended Guidance

The tool runs `ronyup setup workspace` with the provided `path` as the working directory.
The result is a Go workspace containing `go.work`, `cmd/service/`, `pkg/i18n/`, an empty
`feature/` tree, and a `.ai/mcp/mcp.json` for IDE integration.

After scaffolding, add feature modules with the `scaffold_feature` tool, then run
`make tidy && make lint && make test` from the workspace root.
