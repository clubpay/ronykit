#!/usr/bin/env bash
#
# Claude Code `Stop` hook — enforces the backend verify-and-fix loop. When backend
# feature files changed and verify.sh fails, exits 2 so Claude Code keeps the turn
# going with the failure as feedback. Claude Code caps consecutive Stop-hook
# continuations itself, so a gate that cannot pass never loops forever.
#
# FAILS OPEN: no feature modules, nothing changed, or go missing → allow stopping.
set -uo pipefail

cat >/dev/null 2>&1 || true

cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0

# Resolve backend root: fullstack uses backend/, backend-only uses repo root.
backend_root=""
if [ -f backend/verify.sh ]; then
	backend_root="backend"
elif [ -f verify.sh ] && [ -d feature ]; then
	backend_root="."
else
	exit 0
fi

command -v go >/dev/null 2>&1 || exit 0

# Only act when backend feature paths changed.
if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	changed=""
	if [ "$backend_root" = "backend" ]; then
		changed="$(git status --porcelain -- backend/feature backend/go.work 2>/dev/null)"
	else
		changed="$(git status --porcelain -- feature go.work 2>/dev/null)"
	fi
	[ -n "$changed" ] || exit 0
fi

out="$(bash "$backend_root/verify.sh" 2>&1)"
status=$?

if [ "$status" -eq 0 ]; then
	exit 0
fi

printf '%s\n\n%s\n' "Backend verification failed. Fix every issue below yourself — write and RUN integration tests for every repo port method and unit tests for every app method — then keep going until \`bash $backend_root/verify.sh\` passes:" "$out" >&2

exit 2
