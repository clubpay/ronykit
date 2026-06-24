#!/usr/bin/env bash
#
# Cursor `stop` hook — enforces the frontend verify-and-fix loop with no user
# interaction. When frontend files changed in this session and
# `frontend/verify.sh` fails, it returns a followup_message so the agent keeps
# fixing until the gate passes (bounded by loop_limit in hooks.json).
#
# It FAILS OPEN (never blocks): if there is no frontend app, nothing changed,
# or the required tooling (node) is missing, it allows completion unchanged.
set -uo pipefail

cat >/dev/null 2>&1 || true   # consume the stop-event JSON on stdin (unused)

emit_empty() { echo '{}'; exit 0; }

# Only relevant for repos that actually have an initialized frontend app
# (single app at frontend/package.json, or multiple at frontend/<app>/).
[ -d frontend ] || emit_empty
have_app=0
[ -f frontend/package.json ] && have_app=1
for f in frontend/*/package.json; do
	[ -f "$f" ] && have_app=1
done
[ "$have_app" -eq 1 ] || emit_empty

# node is needed both to run the JS tooling and to encode JSON safely.
command -v node >/dev/null 2>&1 || emit_empty

# Only act when frontend files actually changed in this working tree.
if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	[ -n "$(git status --porcelain -- frontend 2>/dev/null)" ] || emit_empty
fi

out="$(bash frontend/verify.sh 2>&1)"
status=$?

if [ "$status" -eq 0 ]; then
	emit_empty
fi

printf '%s' "Frontend verification failed. Fix every issue below yourself — do NOT ask me to run anything or confirm fixes — and keep going until \`bash frontend/verify.sh\` passes:

$out" | node -e 'const fs=require("fs");process.stdout.write(JSON.stringify({followup_message: fs.readFileSync(0,"utf8")}))'

exit 0
