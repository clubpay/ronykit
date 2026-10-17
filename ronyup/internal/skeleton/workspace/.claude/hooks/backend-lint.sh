#!/usr/bin/env bash
#
# Claude Code `PostToolUse` hook (Edit|Write) — runs `ronyup lint` on the package of
# the Go file just edited. When that file re-implements an x/rkit helper, `ronyup`
# exits 2 with the findings on stderr, which Claude Code hands to the agent right
# away instead of at `make verify`.
#
# FAILS OPEN: ronyup missing, non-Go or test files, or packages that do not compile
# yet → silent.
command -v ronyup >/dev/null 2>&1 || exit 0
exec ronyup lint --hook claude-post-edit
