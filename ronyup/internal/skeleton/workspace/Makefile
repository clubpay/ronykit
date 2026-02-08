.PHONY: run test tidy vet lint

MODULE_DIRS := $(shell go list -f '{{.Dir}}' -m all | grep -v '/pkg/mod/')

run:
	@echo "Run service"
	@cd cmd/service && go run .

test:
	@echo "Run tests"
	@for dir in $(MODULE_DIRS); do \
		echo "test $$dir..."; \
		(cd $$dir && go test ./...); \
	done

tidy:
	@echo "Run go mod tidy"
	@for dir in $(MODULE_DIRS); do \
		echo "tidy $$dir..."; \
		(cd $$dir && go mod tidy); \
	done

vet:
	@echo "Run go vet"
	@for dir in $(MODULE_DIRS); do \
		echo "vet $$dir..."; \
		(cd $$dir && go vet ./...); \
	done

lint:
	@echo "Run lint (autofix)"
	@for dir in $(MODULE_DIRS); do \
		echo "lint $$dir..."; \
		(cd $$dir && golangci-lint run --fix ./...); \
	done
