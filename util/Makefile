.PHONY:  test test-short test-race cleanup setup

cleanup:
	@echo "Running cleanup over packages"
	@go mod tidy
	@go fmt ./...
	@golangci-lint run  --fix ./...
	@go vet ./...

test:
	@echo "Run Tests"
	@gotestsum --hide-summary=output --format pkgname-and-test-fails --format-hide-empty-pkg --max-fails 1 -- \
					-parallel 10 -covermode=atomic -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out -o=coverage.out

test-short:
	@echo "Run Tests (Shorts)"
	@gotestsum --hide-summary=output --format pkgname-and-test-fails --format-hide-empty-pkg --max-fails 1 -- \
					-test.short -count=1 -parallel 10 -covermode=atomic -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out -o=coverage.out
