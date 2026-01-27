# ronyup

`ronyup` is a CLI that scaffolds RonyKIT workspaces and features, and supports i18n catalog generation for Go projects.

## Install

From the repo root:

```bash
go install ./ronyup
```

Or with a module path:

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

## Commands

- `setup`: create a new workspace or add a feature to an existing workspace.
- `text`: extract and generate translation catalogs using `golang.org/x/text/message/pipeline`.
- `template`: parse Go source and print package/type/function metadata (generator stub).

Running `ronyup setup` or `ronyup text` with no flags triggers an interactive TUI.

## Usage

### Create a workspace

```bash
ronyup setup workspace \
  --repoDir ./my-repo \
  --repoModule github.com/your/repo
```

This copies the embedded workspace skeleton, initializes a `go.work`, and creates module roots for:

- `pkg/i18n`
- `cmd/service`

### Add a feature

```bash
ronyup setup feature \
  --featureDir services/auth \
  --featureName auth \
  --template service
```

The feature is copied from the embedded templates in `internal/skeleton/feature`.

### Generate translation catalogs

```bash
ronyup text \
  --src-lang en-US \
  --dst-lang en-US,fa-IR,ar-AR \
  --out-dir . \
  --gen-package pkg/i18n \
  --gen-file catalog.go
```

If a `go.work` file is present, the command loads packages from each `use` entry (filtered by `--modules-filter`).

### Inspect Go source

```bash
ronyup template
```

This parses packages under the current directory and prints collected metadata (packages, types, functions, comments).

## Flags

### setup

Global flags:

- `--repoModule`, `-m`: Go module path for the repository.
- `--force`, `-f`: clean destination directory before setup.
- `--custom`, `-c`: custom key/value map for templates.

Workspace flags:

- `--repoDir`, `-r`: destination directory for the workspace.

Feature flags:

- `--featureDir`, `-p`: destination directory inside the repo.
- `--featureName`, `-n`: feature name.
- `--template`, `-t`: feature template (`service`, `job`, `gateway`).

### text

- `--src-lang`, `-s`: source language (default `en-US`).
- `--dst-lang`, `-l`: destination languages (default `en-US,fa-IR,ar-AR`).
- `--out-dir`, `-o`: output directory (default `.`).
- `--gen-package`, `-g`: package/path for generated file.
- `--gen-file`, `-f`: generated file name (default `catalog.go`).
- `--modules-filter`, `-m`: go.work module filter list.

### template

- `--templateRepo`, `-r`: custom templates repo (not used yet).
- `--template`, `-t`: template name (placeholder).
- `--custom`, `-c`: custom key/value map for templates.

## Notes

- Only the `service` feature template is currently included under `internal/skeleton/feature`.
- `template generate` is a stub; the `template` command only inspects Go source today.
- `--force`, `--custom`, and `--templateRepo` are accepted but not wired into behavior yet.
# ronyup

`ronyup` is the CLI for RonyKIT scaffolding and tooling. It can:

- Create a new Go workspace with a RonyKIT service layout.
- Add new feature modules (service/job/gateway) to an existing workspace.
- Extract and generate translation catalogs.
- Inspect Go packages for template-related metadata (WIP).

## Installation

```bash
go install github.com/clubpay/ronykit/ronyup@latest
```

## Usage

```bash
ronyup <command> [subcommand] [flags]
```

Run `ronyup` with no args/flags to enter interactive mode (where supported).

## Commands

### setup

Create a workspace or add a feature module.

```bash
ronyup setup [workspace|feature] [flags]
```

Global flags for `setup`:

```
-m, --repoModule string        go module for the repository (default "github.com/your/repo")
-f, --force                    clean destination directory before setup
-c, --custom stringToString    custom values for the template
```

Notes:
- `--custom` is parsed but not currently applied in the template renderer.
- Running `ronyup setup` with no args/flags starts the interactive prompt.

#### setup workspace

Scaffolds a new workspace and initializes Go workspace/modules.

```bash
ronyup setup workspace -r ./my-repo -m github.com/you/myrepo
```

Flags:

```
-r, --repoDir string    destination directory for the setup (default "./my-repo")
```

What it does:
- Copies the embedded workspace skeleton to `repoDir`.
- Creates `go.work`, initializes `pkg/i18n` and `cmd/service` modules.
- Sets `go` version to `1.25`, runs `go mod tidy` and `go work use .`.
- Initializes a git repo and makes an initial commit.

#### setup feature

Adds a new feature module under `feature/<template>/<featureDir>`.

```bash
ronyup setup feature -m github.com/you/myrepo -p auth -n auth -t service
```

Flags:

```
-p, --featureDir string    destination directory inside repoDir for the setup (default "my_feature")
-n, --featureName string   feature name (default "myfeature")
-t, --template string      possible values: service | job | gateway (default "service")
```

Requirements and side effects:
- Must be run from a Go workspace root (a `go.work` file must exist).
- Writes into `feature/<template>/<featureDir>` in the current directory.
- Initializes a Go module for the feature, sets `go` to `1.25`, runs `go mod tidy`,
  `go fmt`, and adds the module to `go.work`.
- Adds a side-effect import to `cmd/service/features.go` and runs `go mod tidy` + `go fmt`
  in `cmd/service`.
- Use `--force` to replace a non-empty feature directory.

### text

Extracts and generates translation catalogs using `golang.org/x/text/message/pipeline`.

```bash
ronyup text [flags]
```

Flags:

```
-s, --src-lang string         source language (default "en-US")
-l, --dst-lang strings        languages to generate (default ["en-US","fa-IR","ar-AR"])
-o, --out-dir string          output path (default ".")
-g, --gen-package string      package or relative path for generated file
-f, --gen-file string         generated file name (default "catalog.go")
-m, --modules-filter strings  modules filter (go.work paths)
```

Behavior:
- If `go.work` exists, it loads all workspace modules (or those in `--modules-filter`).
- Otherwise it scans the current module.
- Prints detected package list before extraction.
- Runs extract → import → merge → export → generate.

Run `ronyup text` with no args/flags to enter the interactive prompt.

### template

Inspect Go packages for template metadata (WIP).

```bash
ronyup template [flags]
```

Flags:

```
-c, --custom stringToString    custom values for the template
```

Behavior:
- Parses Go packages under the current directory and prints counts of files, types,
  functions, and comments.

#### template generate

```bash
ronyup template generate [flags]
```

Flags:

```
-r, --templateRepo string   if not set we use the standard templates repo embedded into the binary
-t, --template string       possible values: dto | dao | serializer (default "service")
```

Note: the `generate` subcommand is currently a no-op in the source.

## Examples

Create a new workspace:

```bash
ronyup setup workspace -r ./my-repo -m github.com/you/myrepo
```

Add a new service feature:

```bash
ronyup setup feature -m github.com/you/myrepo -p auth -n auth -t service
```

Generate translations:

```bash
ronyup text -s en-US -l en-US,fa-IR -o ./i18n -f catalog.go
```

## Requirements

- Go toolchain available on `PATH`.
- Git available on `PATH` for `setup workspace`.
- A Go workspace root (`go.work`) for `setup feature`.
