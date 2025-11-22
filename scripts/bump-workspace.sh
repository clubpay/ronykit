#!/usr/bin/env bash

# bump-workspace.sh — Bump versions for all Go modules in this workspace
#
# Overview
# - Scans go.work to discover module directories and their module paths.
# - Ignores example modules under "example/*".
# - For each module, finds the latest git tag following the convention "<dir>/vX.Y.Z".
# - Computes the next version per module (minor|patch).
# - Updates go.mod files so that workspace-internal dependencies require the newly computed versions.
# - Creates and pushes tags for each module, and commits/pushes go.mod changes.
# - Supports a dry-run mode that only prints actions without modifying anything.
#
# Usage
#   scripts/bump-workspace.sh [--part patch|minor] [--dry-run]
#
# Notes
# - Version tags are per-module using the directory path as prefix, e.g., "kit/v1.2.3".
# - If a module has no prior tags, it starts from v0.0.0 and bumps accordingly.
# - Requires: bash, git, go, sed, awk.
#   - macOS: compatible with the default Bash 3.2 (no associative arrays, no mapfile)
#   - Windows: run via Git Bash or WSL

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

PART="patch"   # default bump kind
DRY_RUN=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --part)
      PART=${2:-}
      shift 2
      ;;
    --dry-run|-n)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [--part patch|minor] [--dry-run]"; exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2; exit 1
      ;;
  esac
done

if [[ "$PART" != "patch" && "$PART" != "minor" ]]; then
  echo "--part must be 'patch' or 'minor'" >&2
  exit 1
fi

say() {
  echo "[bump] $*"
}

run() {
  if [[ $DRY_RUN -eq 1 ]]; then
    say "DRY-RUN: $*"
  else
    eval "$*"
  fi
}

# Parse go.work to collect module directories
if [[ ! -f go.work ]]; then
  echo "go.work not found at $ROOT_DIR" >&2
  exit 1
fi

# Collect module directories (portable; avoid Bash 4 mapfile and process substitution)
MODULE_DIRS=()
__tmp_mods=$(mktemp 2>/dev/null || echo "/tmp/ronykit_mods_$$")
# Extract module paths from go.work safely (supports both block and single-line use)
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

say "Discovered ${#MODULE_DIRS[@]} module directories from go.work (excluding example/*)"

# Build arrays parallel by index: DIRS[i], PATHS[i], CURRS[i], NEXTS[i]
DIRS=()
PATHS=()
CURRS=()
NEXTS=()

for dir in "${MODULE_DIRS[@]}"; do
  gomod="$dir/go.mod"
  if [[ ! -f "$gomod" ]]; then
    echo "Skipping $dir (no go.mod)" >&2
    continue
  fi
  modpath=$(awk '/^module /{print $2; exit}' "$gomod")
  if [[ -z "$modpath" ]]; then
    echo "Could not read module path from $gomod" >&2
    exit 1
  fi
  DIRS+=("$dir")
  PATHS+=("$modpath")

  # Find latest tag like "<dir>/v*" (use git's version sorting for portability)
  # Some repos tag without a leading "./" (e.g., "flow/vX.Y.Z"), while go.work may list "./flow".
  # Try both variants to detect the existing version correctly.
  dir_no_dot="${dir#./}"
  latest_tag=$(git tag --list "$dir_no_dot/v*" --sort=-version:refname | head -n1 || true)
  if [[ -z "$latest_tag" ]]; then
    latest_tag=$(git tag --list "$dir/v*" --sort=-version:refname | head -n1 || true)
  fi
  if [[ -z "$latest_tag" ]]; then
    CURRS+=("v0.0.0")
  else
    # Extract the vX.Y.Z part (after last slash)
    CURRS+=("${latest_tag##*/}")
  fi
done

# helpers to locate index by dir or module path
idx_by_dir() {
  local target="$1"
  local i
  for i in "${!DIRS[@]}"; do
    if [[ "${DIRS[$i]}" == "$target" ]]; then
      echo "$i"; return 0
    fi
  done
  echo "-1"
}

idx_by_path() {
  local target="$1"
  local i
  for i in "${!PATHS[@]}"; do
    if [[ "${PATHS[$i]}" == "$target" ]]; then
      echo "$i"; return 0
    fi
  done
  echo "-1"
}

increment_version() {
  local ver="$1"    # vX.Y.Z
  local part="$2"   # patch|minor

  local core=${ver#v}
  local major minor patch
  IFS='.' read -r major minor patch <<<"$core"
  if [[ "$part" == "minor" ]]; then
    minor=$((minor+1))
    patch=0
  else
    patch=$((patch+1))
  fi
  echo "v${major}.${minor}.${patch}"
}

# Compute new versions for each module (NEXTS aligned by index)
for i in "${!DIRS[@]}"; do
  curr="${CURRS[$i]}"
  next=$(increment_version "$curr" "$PART")
  NEXTS[$i]="$next"
done

say "Planned versions (part=$PART):"
for i in "${!DIRS[@]}"; do
  printf "  - %s: %s -> %s\n" "${DIRS[$i]}" "${CURRS[$i]}" "${NEXTS[$i]}"
done

# Update go.mod requires for intra-workspace dependencies
changed_modules=()
for dir in "${DIRS[@]}"; do
  gomod="$dir/go.mod"
  [[ -f "$gomod" ]] || continue

  # Read dependency paths from a temporary file to avoid process substitution
  __tmp_deps=$(mktemp 2>/dev/null || echo "/tmp/ronykit_deps_$$")
  awk '/^require \(/ {inb=1; next} /^\)/{inb=0} inb {print $1} /^require /{print $2}' "$gomod" \
    | sed -E 's/[\"\)]//g' \
    | awk 'NF>0{print $1}' > "$__tmp_deps"
  while IFS= read -r dep_path; do
    [[ -z "$dep_path" ]] && continue

    # Find the index that provides this dep_path
    dep_idx=$(idx_by_path "$dep_path")
    if [[ "$dep_idx" == "-1" ]]; then
      continue
    fi

    new_ver="${NEXTS[$dep_idx]}"
    # Determine current required version (if any) by parsing this module's go.mod
    curr_req=$(awk -v m="$dep_path" 'inb && $1==m {print $2} /^require \(/ {inb=1} /^\)/{inb=0} /^require / {if ($2==m) print $3}' "$gomod" | sed -E 's/[\"]//g' | head -n1 || true)

    if [[ -n "$new_ver" ]]; then
      if [[ $DRY_RUN -eq 1 ]]; then
        say "$dir: would set require $dep_path@$new_ver (was ${curr_req:-unset})"
      else
        pushd "$dir" >/dev/null
        go mod edit -require="$dep_path@$new_ver"
        popd >/dev/null
        # append if not already in the list
        local_present=0
        for cm in "${changed_modules[@]:-}"; do
          if [[ "$cm" == "$dir" ]]; then local_present=1; break; fi
        done
        if [[ $local_present -eq 0 ]]; then
          changed_modules+=("$dir")
        fi
        say "$dir: set require $dep_path@$new_ver (was ${curr_req:-unset})"
      fi
    fi
  done < "$__tmp_deps"
  rm -f "$__tmp_deps"

  if [[ $DRY_RUN -eq 0 ]]; then
    if [[ -d "$dir" ]]; then
      pushd "$dir" >/dev/null
      run "go mod tidy"
      popd >/dev/null
    fi
  else
    say "$dir: would run go mod tidy"
  fi
done

# changed_modules already kept unique in the loop above (portable, no readarray)

# Commit go.mod changes (if any)
if [[ ${#changed_modules[@]} -gt 0 ]]; then
  msg="bump workspace requires ($PART)"
  if [[ $DRY_RUN -eq 1 ]]; then
    say "Would git add go.mod and go.sum in: ${changed_modules[*]}"
    say "Would commit with message: $msg"
    say "Would push commit"
  else
    for d in "${changed_modules[@]}"; do
      run "git add $d/go.mod || true"
      run "git add $d/go.sum || true"
    done
    run "git commit -m '$msg' || true"
    run "git push || true"
  fi
else
  say "No go.mod changes detected"
fi

# Create and push tags for each module
for i in "${!DIRS[@]}"; do
  dir="${DIRS[$i]}"
  new_ver="${NEXTS[$i]}"
  tag="$dir/$new_ver"
  if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
    say "Tag already exists: $tag (skipping)"
    continue
  fi
  if [[ $DRY_RUN -eq 1 ]]; then
    say "Would create annotated tag: $tag"
  else
    run "git tag -a '$tag' -m '$new_ver'"
  fi
done

if [[ $DRY_RUN -eq 1 ]]; then
  say "Would push tags"
else
  run "git push --tags"
fi

say "Done."
