#!/usr/bin/env bash

# github-release.sh — Create a single GitHub release whose description lists
# every workspace module and its latest tag.
#
# Discovers modules from go.work (excluding examples), finds each module's
# latest git tag, builds a markdown body, and creates one GitHub release.
# Supports --dry-run to preview without creating the release.
#
# Usage
#   scripts/github-release.sh --tag <tag> [--dry-run]
#
# Example
#   scripts/github-release.sh --tag v0.24.0
#   scripts/github-release.sh --tag v0.24.0 --dry-run
#
# Requires: bash, git, gh (GitHub CLI), awk, sed

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

DRY_RUN=0
TAG=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag)
      TAG="${2:-}"
      shift 2
      ;;
    --dry-run|-n)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      echo "Usage: $0 --tag <tag> [--dry-run]"; exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2; exit 1
      ;;
  esac
done

if [[ -z "$TAG" ]]; then
  echo "--tag is required" >&2
  exit 1
fi

say() {
  echo "[github-release] $*"
}

# Parse go.work to collect module directories (same logic as bump-workspace.sh)
if [[ ! -f go.work ]]; then
  echo "go.work not found at $ROOT_DIR" >&2
  exit 1
fi

MODULE_DIRS=()
__tmp_mods=$(mktemp 2>/dev/null || echo "/tmp/ronykit_mods_$$")
awk '
  BEGIN{inb=0}
  /^use[[:space:]]*\(/{inb=1; next}
  inb && /^\)/{inb=0; next}
  inb {print $1; next}
  /^use[[:space:]]/ {print $2}
' go.work \
  | sed -E 's/^[[:space:]]*//; s/[\"\)]//g' \
  | awk 'NF>0 {print $1}' \
  | awk '!match($0, /^(\.\/)?example(\/|$)/)' > "$__tmp_mods"
while IFS= read -r line; do
  [ -z "$line" ] && continue
  MODULE_DIRS+=("$line")
done < "$__tmp_mods"
rm -f "$__tmp_mods"

if [[ ${#MODULE_DIRS[@]} -eq 0 ]]; then
  echo "No module directories discovered from go.work" >&2
  exit 1
fi

say "Discovered ${#MODULE_DIRS[@]} modules from go.work (excluding example/*)"

# Build the release body: a table of modules and their latest tags
body="## Released Modules"
body+=$'\n\n'
body+="| Module | Version |"
body+=$'\n'
body+="|--------|---------|"
body+=$'\n'

for dir in "${MODULE_DIRS[@]}"; do
  dir_no_dot="${dir#./}"

  latest_tag=$(git tag --list "$dir_no_dot/v*" --sort=-version:refname | head -n1 || true)
  if [[ -z "$latest_tag" ]]; then
    latest_tag=$(git tag --list "$dir/v*" --sort=-version:refname | head -n1 || true)
  fi

  if [[ -z "$latest_tag" ]]; then
    body+="| \`$dir_no_dot\` | _no tags_ |"
  else
    version="${latest_tag##*/}"
    body+="| \`$dir_no_dot\` | \`$version\` |"
  fi
  body+=$'\n'
done

say "Release tag: $TAG"
say "Release body:"
echo "$body"

if [[ $DRY_RUN -eq 1 ]]; then
  say "DRY-RUN: would create release for tag $TAG"
else
  say "Creating release for $TAG ..."
  gh release create "$TAG" --title "$TAG" --notes "$body"
  say "Done."
fi
