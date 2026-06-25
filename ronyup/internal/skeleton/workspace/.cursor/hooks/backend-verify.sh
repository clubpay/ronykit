#!/usr/bin/env bash
#
# Cursor `stop` hook — enforces the backend verify-and-fix loop. When backend
# feature files changed and verify.sh fails, returns followup_message so the
# agent keeps fixing until the gate passes (bounded by loop_limit in hooks.json).
#
# FAILS OPEN: no feature modules, nothing changed, or go missing → allow completion.
set -uo pipefail

cat >/dev/null 2>&1 || true

emit_empty() { echo '{}'; exit 0; }

# Resolve backend root: fullstack uses backend/, backend-only uses repo root.
backend_root=""
if [ -f backend/verify.sh ]; then
	backend_root="backend"
elif [ -f verify.sh ] && [ -d feature ]; then
	backend_root="."
else
	emit_empty
fi

command -v go >/dev/null 2>&1 || emit_empty

# Only act when backend feature paths changed.
if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	changed=""
	if [ "$backend_root" = "backend" ]; then
		changed="$(git status --porcelain -- backend/feature backend/go.work 2>/dev/null)"
	else
		changed="$(git status --porcelain -- feature go.work 2>/dev/null)"
	fi
	[ -n "$changed" ] || emit_empty
fi

out="$(bash "$backend_root/verify.sh" 2>&1)"
status=$?

if [ "$status" -eq 0 ]; then
	emit_empty
fi

printf '%s' "Backend verification failed. Fix every issue below yourself — write and RUN integration tests for every repo port method and unit tests for every app method — then keep going until \`bash %s/verify.sh\` passes:

%s" "$backend_root" "$out" | node -e 'const fs=require("fs");process.stdout.write(JSON.stringify({followup_message: fs.readFileSync(0,"utf8")}))'

exit 0
