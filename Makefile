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

vet:
	@echo "Run Go Vet"
	@for dir in $$(go list -f '{{.Dir}}' -m all | grep -v mod | grep -v example); do \
		echo "Go Vet $$dir..."; \
		(cd $$dir && go vet ./...); \
	done

test:
	@echo "Run Tests"
	@for dir in $$(go list -f '{{.Dir}}' -m all | grep -v mod | grep -v example); do \
		echo "Go Test $$dir..."; \
		(cd $$dir && gotestsum --hide-summary=output --format pkgname-and-test-fails \
                   					--format-hide-empty-pkg --max-fails 1 \
                   					-- -covermode=atomic -coverprofile=coverage.out ./... \
                   				); \
	done

new-version-minor:
	@bash ./scripts/update-version.sh kit 1

