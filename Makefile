.PHONY:  cleanup increment-version-patch increment-version-minor

cleanup:
	@echo "Cleanup"
	@bash ./scripts/cleanup.sh

increment-version-patch:
	@bash ./scripts/update-version.sh kit 2

increment-version-minor:
	@bash ./scripts/update-version.sh kit 1

