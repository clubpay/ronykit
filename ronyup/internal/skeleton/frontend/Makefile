.PHONY: verify verify-no-stories

# verify is the frontend quality gate: typecheck, lint, build, test,
# build-storybook, and stories-coverage for every app under frontend/.
# It is the same gate the Cursor stop hook runs, and what CI should run.
verify:
	@bash verify.sh

# Same gate without the Storybook stories-coverage check.
verify-no-stories:
	@SKIP_STORIES_CHECK=1 bash verify.sh
