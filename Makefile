.PHONY:  cleanup test lint vet format-md \
         new-version-patch new-version-minor \
         new-version-patch-dry new-version-minor-dry \
         bump-workspace bump-workspace-dry \
         bump-module bump-module-dry \
         ronyup-release homebrew-release

setup:
	@echo "Install required tools"
	@go install gotest.tools/gotestsum@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/Kunde21/markdownfmt/v3/cmd/markdownfmt@latest

format-md:
	@echo "Format Markdown files"
	@bash ./scripts/format-markdown.sh

lint:
	@echo "Run Linting"
	@for dir in $$(go list -f '{{.Dir}}' -m all | grep -v mod | grep -v example); do \
		echo "lint $$dir..."; \
		(cd $$dir && golangci-lint run --path-prefix "${dir}" --fix ./...); \
	done

vet:
	@echo "Run Go Vet"
	@for dir in $$(go list -f '{{.Dir}}' -m all | grep -v mod | grep -v example); do \
		echo "go vet $$dir..."; \
		(cd $$dir && go vet ./...); \
	done

tidy:
	@echo "Run Go Mod Tidy"
	@for dir in $$(go list -f '{{.Dir}}' -m all | grep -v mod | grep -v example); do \
		echo "go mod tidy $$dir..."; \
		(cd $$dir && go mod tidy ); \
	done

test:
	@echo "Run Tests"
	@for dir in $$(go list -f '{{.Dir}}' -m all | grep -v mod | grep -v example | grep -v ronyup); do \
		echo "test $$dir..."; \
		(cd $$dir && gotestsum --hide-summary=output --format pkgname-and-test-fails \
                   					--format-hide-empty-pkg --max-fails 1 \
                   					-- -covermode=atomic -coverprofile=coverage.out ./... \
                   				); \
		done

# Workspace version bump helpers (uses scripts/bump-workspace.sh)
# Usage:
#   make bump-workspace PART=minor              # apply changes to all modules
#   make bump-workspace-dry PART=patch          # dry-run only
#   make bump-module MODULE=ronyup PART=minor   # bump a single module
#   make bump-module-dry MODULE=ronyup          # dry-run a single module

PART ?= patch

bump-workspace:
	@bash ./scripts/bump-workspace.sh --part $(PART)

bump-workspace-dry:
	@bash ./scripts/bump-workspace.sh --part $(PART) --dry-run

bump-module:
	@test -n "$(MODULE)" || { echo "MODULE is required, e.g. make bump-module MODULE=ronyup"; exit 1; }
	@bash ./scripts/bump-workspace.sh --part $(PART) --module $(MODULE)

bump-module-dry:
	@test -n "$(MODULE)" || { echo "MODULE is required, e.g. make bump-module-dry MODULE=ronyup"; exit 1; }
	@bash ./scripts/bump-workspace.sh --part $(PART) --module $(MODULE) --dry-run

github-release:
	@bash ./scripts/github-release.sh --tag $(TAG)

github-release-dry:
	@bash ./scripts/github-release.sh --tag $(TAG) --dry-run

# Trigger the "RonyUP Release" workflow (build binaries, publish release, bump tap).
# Override the tag with: make ronyup-release TAG=ronyup/v0.4.7
ronyup-release:
	@command -v gh >/dev/null 2>&1 || { echo "gh (GitHub CLI) is required"; exit 1; }
	@tag="$(TAG)"; \
	if [ -z "$$tag" ]; then \
		tag=$$(git tag --list 'ronyup/v*' --sort=-version:refname | head -n1); \
	fi; \
	if [ -z "$$tag" ]; then echo "No ronyup/v* tag found"; exit 1; fi; \
	echo "Triggering RonyUP release for $$tag"; \
	gh workflow run ronyup-release.yml -f tag="$$tag"

# Re-run only the Homebrew formula bump for an existing release (no rebuild).
# Override the tag with: make homebrew-release TAG=ronyup/v0.4.7
homebrew-release:
	@command -v gh >/dev/null 2>&1 || { echo "gh (GitHub CLI) is required"; exit 1; }
	@tag="$(TAG)"; \
	if [ -z "$$tag" ]; then \
		tag=$$(git tag --list 'ronyup/v*' --sort=-version:refname | head -n1); \
	fi; \
	if [ -z "$$tag" ]; then echo "No ronyup/v* tag found"; exit 1; fi; \
	echo "Triggering Homebrew formula bump for $$tag"; \
	gh workflow run homebrew.yml -f tag="$$tag"

################
# RonyUP
#

dev-mcp:
	@go install ./ronyup
	@npx @modelcontextprotocol/inspector ronyup mcp
