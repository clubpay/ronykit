#!/usr/bin/env bash
#
# Frontend verify gate — the single source of truth for "frontend is done".
#
# For every frontend app (this directory for a single app, or each
# frontend/<app>/ for multiple apps) it runs the quality steps that the app
# actually defines (typecheck, lint, build, test, build-storybook), checks UI
# polish/a11y conventions, and checks that each UI component ships a co-located
# Storybook story. It runs ONLY the npm scripts present in each app's
# package.json, so it is framework-agnostic.
#
# It exits non-zero if any step fails, so agents and CI can loop
# "fix -> re-run" until everything is green. A fresh, un-initialized frontend
# (no package.json) is a no-op success.
#
# Usage:
#   bash verify.sh                         # verify every app under frontend/
#   SKIP_STORIES_CHECK=1 bash verify.sh    # skip the stories-coverage check
#   SKIP_UI_POLISH_CHECK=1 bash verify.sh  # skip the UI polish/a11y static check
set -uo pipefail

FRONTEND_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# pm_for <app_dir> — pick the package manager based on the lockfile.
pm_for() {
	if [ -f "$1/pnpm-lock.yaml" ]; then echo pnpm
	elif [ -f "$1/yarn.lock" ]; then echo yarn
	elif [ -f "$1/bun.lockb" ]; then echo bun
	else echo npm
	fi
}

# has_script <app_dir> <name> — 0 if package.json defines that npm script.
has_script() {
	node -e '
		const fs = require("fs");
		try {
			const j = JSON.parse(fs.readFileSync(process.argv[1] + "/package.json", "utf8"));
			process.exit(j.scripts && j.scripts[process.argv[2]] ? 0 : 1);
		} catch (e) { process.exit(1); }
	' "$1" "$2"
}

# run_script <app_dir> <script>
run_script() {
	local dir="$1" script="$2" pm
	pm="$(pm_for "$dir")"
	echo ">> ($dir) $pm run $script"
	( cd "$dir" && "$pm" run "$script" )
}

# check_stories <app_dir> — every PascalCase component under a components/
# directory must have a co-located <Name>.stories.tsx. Only enforced once
# Storybook is configured in the app (otherwise it would be a chicken-and-egg).
check_stories() {
	local app="$1"

	if [ ! -d "$app/.storybook" ] && ! has_script "$app" "storybook" && ! has_script "$app" "build-storybook"; then
		echo ">> ($app) Storybook not configured yet — skipping stories-coverage check."
		return 0
	fi

	local missing="" f dir base story
	while IFS= read -r f; do
		[ -z "$f" ] && continue
		dir="$(dirname "$f")"
		base="$(basename "$f" .tsx)"
		story="$dir/$base.stories.tsx"
		[ -f "$story" ] || missing="$missing\n  - $f (expected $base.stories.tsx)"
	done <<EOF
$(find "$app" -type d -name node_modules -prune -o -type f -name '[A-Z]*.tsx' -path '*components*' -print | grep -Ev '\.stories\.tsx$|\.test\.tsx$|\.spec\.tsx$')
EOF

	if [ -n "$missing" ]; then
		printf ">> (%s) Components missing a co-located Storybook story:%b\n" "$app" "$missing"
		return 1
	fi

	echo ">> ($app) stories-coverage OK."
	return 0
}

# check_ui_polish <app_dir> — mechanical check for high-confidence UI/motion/a11y
# anti-patterns from the ui-polish and ux-quality skills. Silence a deliberate
# line with `ui-polish-ignore`.
check_ui_polish() {
	local app="$1"
	node -e '
		const fs = require("fs");
		const path = require("path");
		const root = process.argv[1];
		const skipDirs = new Set(["node_modules", ".next", "dist", "build", "storybook-static", ".git", "coverage"]);
		const findings = [];

		function walk(dir) {
			let entries;
			try { entries = fs.readdirSync(dir, { withFileTypes: true }); } catch { return; }
			for (const e of entries) {
				if (skipDirs.has(e.name)) continue;
				const full = path.join(dir, e.name);
				if (e.isDirectory()) { walk(full); continue; }
				if (!/\.(tsx|jsx|css)$/.test(e.name)) continue;
				if (/\.(stories|test|spec)\.(tsx|jsx)$/.test(e.name)) continue;
				checkFile(full);
			}
		}

		function checkFile(file) {
			const rel = path.relative(root, file);
			const lines = fs.readFileSync(file, "utf8").split("\n");
			lines.forEach((line, idx) => {
				if (line.includes("ui-polish-ignore")) return;
				const loc = `${rel}:${idx + 1}`;
				if (/\btransition-all\b|transition\s*:\s*all\b/.test(line)) {
					findings.push(`${loc}: avoid transition-all / transition: all — name exact properties (e.g. transition-[transform,opacity] or transition-colors) [ui-polish]`);
				}
				if (/\b(?:min-h-screen|h-screen|100vh)\b/.test(line)) {
					findings.push(`${loc}: prefer 100dvh / min-h-dvh / h-dvh over 100vh / h-screen so mobile browser toolbars do not clip layouts [ux-quality]`);
				}
				if (/\bw-screen\b/.test(line)) {
					findings.push(`${loc}: prefer w-full (width: 100%) over w-screen (100vw causes horizontal scrollbar overflow) [ux-quality]`);
				}
				if (/\b(?:scale-0|scale\(0\))(?![.\d])/.test(line)) {
					findings.push(`${loc}: never animate from scale(0) / scale-0 — start entrances from scale(0.95) + opacity: 0 [ui-polish]`);
				}
			});
		}

		walk(root);
		if (findings.length > 0) {
			console.error(`>> (${root}) UI polish / UX quality violations (${findings.length}):`);
			for (const f of findings) console.error("  - " + f);
			process.exit(1);
		}
		console.log(`>> (${root}) UI polish static check OK.`);
	' "$app"
}

command -v node >/dev/null 2>&1 || {
	echo "node is required to run the frontend verify gate but was not found on PATH." >&2
	exit 1
}

# design_doc_for <app_dir> — print path to required design doc (repo-root relative).
design_doc_for() {
	local app="$1" slug
	if [ "$app" = "$FRONTEND_DIR" ]; then
		slug="web"
	else
		slug="$(basename "$app")"
	fi
	echo "$(dirname "$FRONTEND_DIR")/docs/design/${slug}-frontend-design.md"
}

# check_design_doc <app_dir> — require approved frontend design doc once app exists.
check_design_doc() {
	local app="$1" doc
	doc="$(design_doc_for "$app")"
	if [ ! -f "$doc" ]; then
		echo ">> ($app) MISSING design doc: $doc (write with MCP prompt design-frontend; user must approve)" >&2
		return 1
	fi
	if ! grep -qE '^status:[[:space:]]*approved' "$doc"; then
		echo ">> ($app) design doc must have frontmatter status: approved — $doc" >&2
		return 1
	fi
	echo ">> ($app) design doc OK ($doc)."
	return 0
}

# Discover apps.
apps=""
if [ -f "$FRONTEND_DIR/package.json" ]; then
	apps="$FRONTEND_DIR"
else
	for d in "$FRONTEND_DIR"/*/; do
		[ -f "${d}package.json" ] && apps="$apps ${d%/}"
	done
fi

if [ -z "$apps" ]; then
	echo "No frontend app found (no package.json under $FRONTEND_DIR) — nothing to verify."
	exit 0
fi

fail=0
for app in $apps; do
	echo "=== Verifying app: $app ==="

	check_design_doc "$app" || fail=1

	if [ ! -d "$app/node_modules" ]; then
		pm="$(pm_for "$app")"
		echo ">> ($app) $pm install"
		( cd "$app" && "$pm" install ) || { fail=1; continue; }
	fi

	for step in typecheck lint build test; do
		if has_script "$app" "$step"; then
			run_script "$app" "$step" || fail=1
		fi
	done

	if has_script "$app" "build-storybook"; then
		run_script "$app" "build-storybook" || fail=1
	fi

	if [ "${SKIP_UI_POLISH_CHECK:-0}" != "1" ]; then
		check_ui_polish "$app" || fail=1
	fi

	if [ "${SKIP_STORIES_CHECK:-0}" != "1" ]; then
		check_stories "$app" || fail=1
	fi
done

if [ "$fail" -ne 0 ]; then
	echo "Frontend verify FAILED — fix the issues above and re-run." >&2
	exit 1
fi

echo "Frontend verify PASSED."
exit 0
