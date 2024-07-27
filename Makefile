.PHONY:  cleanup test new-version-patch new-version-minor

setup:
	@echo "Install required tools"
	@go install gotest.tools/gotestsum@latest

cleanup:
	@echo "Cleanup"
	@bash ./scripts/cleanup.sh

test:
	@echo "Running tests"
	@bash ./scripts/run-test.sh

new-version-patch:
	@bash ./scripts/update-version.sh kit 2

new-version-minor:
	@bash ./scripts/update-version.sh kit 1

