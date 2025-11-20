.PHONY:  cleanup test new-version-patch new-version-minor

setup:
	@echo "Install required tools"
	@go install gotest.tools/gotestsum@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	@echo "Run Linting"
	@for dir in $$(go list -f '{{.Dir}}' -m all | grep -v mod | grep -v example); do \
		echo "Linting $$dir..."; \
		(cd $$dir && golangci-lint run --path-prefix "${dir}" --fix ./...); \
	done


test:
	@echo "Run Tests"
	@for dir in $$(grep -oP '(?<=\t\./).*(?=$$)' go.work); do \
		echo "Testing $$dir..."; \
		(cd $$dir && gotestsum ./...); \
	done


new-version-minor:
	@bash ./scripts/update-version.sh kit 1

