#!/usr/bin/env bash
#
# Claude Code `Stop` hook — enforces the frontend verify-and-fix loop with no
# user interaction. When frontend files changed in this session and
# `frontend/verify.sh` fails, exits 2 so Claude Code keeps the turn going with
# the failure as feedback.
#
# It FAILS OPEN (never blocks): if there is no frontend app, nothing changed,
# or the required tooling (node) is missing, it allows completion unchanged.
set -uo pipefail

cat >/dev/null 2>&1 || true

cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0

[ -d frontend ] || exit 0
have_app=0
[ -f frontend/package.json ] && have_app=1
for f in frontend/*/package.json; do
	[ -f "$f" ] && have_app=1
done
[ "$have_app" -eq 1 ] || exit 0

command -v node >/dev/null 2>&1 || exit 0

if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	[ -n "$(git status --porcelain -- frontend 2>/dev/null)" ] || exit 0
fi

out="$(bash frontend/verify.sh 2>&1)"
status=$?

if [ "$status" -eq 0 ]; then
	exit 0
fi

printf '%s\n\n%s\n' "Frontend verification failed. Fix every issue below yourself — do NOT ask me to run anything or confirm fixes — and keep going until \`bash frontend/verify.sh\` passes:" "$out" >&2

exit 2
