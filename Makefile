.PHONY:  cleanup test lint vet \
         new-version-patch new-version-minor \
         new-version-patch-dry new-version-minor-dry \
         bump-workspace bump-workspace-dry

setup:
	@echo "Install required tools"
	@go install gotest.tools/gotestsum@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

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
#   make bump-workspace PART=minor        # apply changes
#   make bump-workspace-dry PART=patch    # dry-run only

PART ?= patch

bump-workspace:
	@bash ./scripts/bump-workspace.sh --part $(PART)

bump-workspace-dry:
	@bash ./scripts/bump-workspace.sh --part $(PART) --dry-run

github-release:
	@gh release create "$(TAG)" --title "$(TITLE)"
