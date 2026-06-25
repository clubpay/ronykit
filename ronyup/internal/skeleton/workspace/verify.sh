#!/usr/bin/env bash
#
# Backend verify gate — the single source of truth for "backend feature work is done".
#
# For every feature module under feature/ it checks:
#   - repo ports with interface methods have integration_test/ with real tests
#   - each repo port method name appears in integration tests
#   - internal/app exported methods have unit tests
#   - internal/app unit tests pass `go test` (no Docker needed)
#   - integration_test packages pass `go test` (only when Docker is available)
#
# Static coverage checks ALWAYS block. The repo integration test RUN needs a
# Docker daemon (Gnomock); when Docker is unavailable the gate WARNS and skips
# only that run, so it never hard-fails for an environment reason.
#
# Exits 0 when there is nothing to check (no feature modules with repo ports).
# Exits non-zero on any failure so agents and CI can loop fix -> re-run.
#
# Usage:
#   bash verify.sh
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

command -v go >/dev/null 2>&1 || {
	echo "go is required to run the backend verify gate but was not found on PATH." >&2
	exit 1
}

# docker_available — integration tests spin up Gnomock containers, which need a
# reachable Docker daemon. When Docker is missing the gate WARNS and skips the
# test RUN (static coverage is still enforced), so the gate never hard-fails for
# a reason outside the agent's control.
docker_available() {
	command -v docker >/dev/null 2>&1 || return 1
	docker info >/dev/null 2>&1 || return 1
	return 0
}

DOCKER_OK=0
if docker_available; then
	DOCKER_OK=1
else
	echo ">> WARNING: Docker is not available — repo integration tests will NOT be executed (Gnomock needs Docker)." >&2
	echo ">> Static coverage is still enforced. Run 'make verify' with Docker running before merging." >&2
fi

fail=0

# repo_methods <port.go> — interface method names (one per line).
repo_methods() {
	awk '
		/interface \{/ { in_iface=1; next }
		in_iface && /^\}/ { in_iface=0; next }
		in_iface && /^[[:space:]]+[A-Z][A-Za-z0-9_]*\(/ {
			name=$1
			sub(/\(.*/, "", name)
			print name
		}
	' "$1"
}

# app_methods <app_dir> — exported App receiver methods (one per line).
app_methods() {
	grep -rhE '^func \(a \*?App\) [A-Z][A-Za-z0-9_]*\(' "$1"/*.go 2>/dev/null \
		| sed -E 's/^func \(a \*?App\) ([A-Z][A-Za-z0-9_]*)\(.*/\1/' \
		| sort -u
}

# test_files_contain <pattern> <files...>
test_files_contain() {
	local needle="$1"
	shift
	local f
	for f in "$@"; do
		[ -f "$f" ] || continue
		if grep -q "$needle" "$f"; then
			return 0
		fi
	done
	return 1
}

# find_feature_modules — directories with go.mod directly under feature/* or feature/*/*
find_feature_modules() {
	find feature -name go.mod 2>/dev/null | while read -r gomod; do
		dirname "$gomod"
	done
}

modules="$(find_feature_modules)"
if [ -z "$modules" ]; then
	echo "No feature modules under feature/ — nothing to verify."
	exit 0
fi

for mod in $modules; do
	echo "=== Verifying module: $mod ==="

	port="$mod/internal/repo/port.go"
	if [ ! -f "$port" ]; then
		echo ">> ($mod) no internal/repo/port.go — skipping repo checks."
		continue
	fi

	methods="$(repo_methods "$port")"
	if [ -z "$methods" ]; then
		echo ">> ($mod) no repository interface methods — skipping repo checks."
	else
		intdir="$mod/internal/repo/integration_test"
		if [ ! -d "$intdir" ]; then
			echo ">> ($mod) MISSING internal/repo/integration_test/ (required for repo ports)" >&2
			fail=1
			continue
		fi

		int_tests=""
		while IFS= read -r tf; do
			[ "$(basename "$tf")" = "setup_test.go" ] && continue
			int_tests="$int_tests $tf"
		done <<EOF
$(find "$intdir" -name '*_test.go' 2>/dev/null)
EOF

		if [ -z "$(echo "$int_tests" | tr -d ' ')" ]; then
			echo ">> ($mod) integration_test/ has no test files (only setup_test.go or empty)" >&2
			fail=1
		else
			for method in $methods; do
				if ! test_files_contain "$method" $int_tests; then
					echo ">> ($mod) repo method $method has no integration test referencing it" >&2
					fail=1
				fi
			done
		fi

		if [ "$DOCKER_OK" -eq 1 ]; then
			echo ">> ($mod) go test ./internal/repo/integration_test/..."
			if ! ( cd "$mod" && go test ./internal/repo/integration_test/... -count=1 ); then
				fail=1
			fi
		else
			echo ">> ($mod) skipping integration test RUN — Docker unavailable (see warning above)." >&2
		fi
	fi

	appdir="$mod/internal/app"
	if [ -d "$appdir" ]; then
		amethods="$(app_methods "$appdir")"
		if [ -n "$amethods" ]; then
			app_tests=""
			while IFS= read -r tf; do
				app_tests="$app_tests $tf"
			done <<EOF
$(find "$appdir" -name '*_test.go' 2>/dev/null)
EOF
			if [ -z "$(echo "$app_tests" | tr -d ' ')" ]; then
				echo ">> ($mod) internal/app has methods but no *_test.go files" >&2
				fail=1
			else
				for method in $amethods; do
					if ! test_files_contain "$method" $app_tests; then
						echo ">> ($mod) app method $method has no unit test referencing it" >&2
						fail=1
					fi
				done

				# App unit tests use fakes at the port boundary, so they run
				# without Docker — always execute them.
				echo ">> ($mod) go test ./internal/app/..."
				if ! ( cd "$mod" && go test ./internal/app/... -count=1 ); then
					fail=1
				fi
			fi
		fi
	fi
done

if [ "$fail" -ne 0 ]; then
	echo "Backend verify FAILED — fix the issues above and re-run." >&2
	exit 1
fi

echo "Backend verify PASSED."
exit 0
