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
# Dependency order (kit now requires x/rkit and x/p)
# - Foundation modules have no intra-workspace requires (x/rkit, x/p, x/settings, …).
#   They are tagged and pushed first so dependents can resolve the new versions.
# - Core modules depend only on foundation (kit, x/batch, x/cache, …). Their
#   go.mod/go.sum are tidied after foundation tags exist, then they are tagged.
# - Remaining modules (rony, stub, gateways, …) are tagged next and tidied last.
# - Post-tag tidy uses GOPRIVATE=github.com/clubpay so freshly pushed tags are
#   fetched from git and do not wait for sum.golang.org / proxy.golang.org.
#
# Usage
#   scripts/bump-workspace.sh [--part patch|minor] [--module <dir>] [--dry-run]
#
# - --module restricts the bump to a single workspace module (e.g. "ronyup" or
#   "std/llms/ollama"). When set, only that module's tag is created/pushed and
#   only its go.mod intra-workspace requires are considered.
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
MODULE_FILTER=""   # when set, restrict bump to this single module dir
# Fresh workspace tags are not on sum.golang.org yet; fetch them from git.
TIDY_ENV="GOPRIVATE=github.com/clubpay"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --part)
      PART=${2:-}
      shift 2
      ;;
    --module|-m)
      MODULE_FILTER=${2:-}
      shift 2
      ;;
    --dry-run|-n)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [--part patch|minor] [--module <dir>] [--dry-run]"; exit 0
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
ALL_MODULE_DIRS=()
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
  ALL_MODULE_DIRS+=("$line")
done < "$__tmp_mods"
rm -f "$__tmp_mods"

if [[ ${#ALL_MODULE_DIRS[@]} -eq 0 ]]; then
  echo "No module directories discovered from go.work" >&2
  exit 1
fi

MODULE_DIRS=("${ALL_MODULE_DIRS[@]}")

# When a single module is requested, narrow MODULE_DIRS to just that one.
# Accept both "ronyup" and "./ronyup" spellings.
if [[ -n "$MODULE_FILTER" ]]; then
  want="${MODULE_FILTER#./}"
  FILTERED=()
  for d in "${MODULE_DIRS[@]}"; do
    if [[ "${d#./}" == "$want" ]]; then
      FILTERED+=("$d")
    fi
  done
  if [[ ${#FILTERED[@]} -eq 0 ]]; then
    echo "Module '$MODULE_FILTER' not found in go.work (or is an excluded example)" >&2
    exit 1
  fi
  MODULE_DIRS=("${FILTERED[@]}")
  say "Restricting bump to single module: ${MODULE_DIRS[0]}"
fi

say "Discovered ${#ALL_MODULE_DIRS[@]} module directories from go.work (excluding example/*)"

# Full workspace catalog (used to classify foundation vs dependents).
ALL_DIRS=()
ALL_PATHS=()
for dir in "${ALL_MODULE_DIRS[@]}"; do
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
  ALL_DIRS+=("$dir")
  ALL_PATHS+=("$modpath")
done

# Build arrays parallel by index for the bump set: DIRS[i], PATHS[i], CURRS[i], NEXTS[i]
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

all_idx_by_path() {
  local target="$1"
  local i
  for i in "${!ALL_PATHS[@]}"; do
    if [[ "${ALL_PATHS[$i]}" == "$target" ]]; then
      echo "$i"; return 0
    fi
  done
  echo "-1"
}

# Print intra-workspace require paths for a module directory (full catalog).
workspace_deps_of() {
  local dir="$1"
  local gomod="$dir/go.mod"
  [[ -f "$gomod" ]] || return 0

  local __tmp_deps
  __tmp_deps=$(mktemp 2>/dev/null || echo "/tmp/ronykit_wsdeps_$$")
  awk '/^require \(/ {inb=1; next} /^\)/{inb=0} inb {print $1} /^require /{print $2}' "$gomod" \
    | sed -E 's/[\"\)]//g' \
    | awk 'NF>0{print $1}' > "$__tmp_deps"
  while IFS= read -r dep_path; do
    [[ -z "$dep_path" ]] && continue
    if [[ "$(all_idx_by_path "$dep_path")" != "-1" ]]; then
      echo "$dep_path"
    fi
  done < "$__tmp_deps"
  rm -f "$__tmp_deps"
}

is_foundation_dir() {
  local dir="$1"
  local __tmp
  __tmp=$(mktemp 2>/dev/null || echo "/tmp/ronykit_found_$$")
  workspace_deps_of "$dir" > "$__tmp"
  if [[ -s "$__tmp" ]]; then
    rm -f "$__tmp"
    return 1
  fi
  rm -f "$__tmp"
  return 0
}

is_core_dir() {
  local dir="$1"
  local __tmp dep dep_idx dep_dir
  __tmp=$(mktemp 2>/dev/null || echo "/tmp/ronykit_core_$$")
  workspace_deps_of "$dir" > "$__tmp"
  if [[ ! -s "$__tmp" ]]; then
    rm -f "$__tmp"
    return 1
  fi
  while IFS= read -r dep; do
    [[ -z "$dep" ]] && continue
    dep_idx=$(all_idx_by_path "$dep")
    dep_dir="${ALL_DIRS[$dep_idx]}"
    if ! is_foundation_dir "$dep_dir"; then
      rm -f "$__tmp"
      return 1
    fi
  done < "$__tmp"
  rm -f "$__tmp"
  return 0
}

append_unique() {
  # $1 = name of array to append to, $2 = value
  local arr_name="$1"
  local value="$2"
  local existing
  eval "local items=(\"\${${arr_name}[@]:-}\")"
  for existing in "${items[@]:-}"; do
    if [[ "$existing" == "$value" ]]; then
      return 0
    fi
  done
  eval "${arr_name}+=(\"\$value\")"
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

create_tag() {
  local dir="$1"
  local new_ver="$2"
  local dir_no_dot="${dir#./}"
  local tag="$dir_no_dot/$new_ver"
  if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
    say "Tag already exists: $tag (skipping)"
    return 0
  fi
  if [[ $DRY_RUN -eq 1 ]]; then
    say "Would create annotated tag: $tag"
  else
    run "git tag -a '$tag' -m '$new_ver'"
  fi
}

tidy_dir() {
  local dir="$1"
  if [[ $DRY_RUN -eq 1 ]]; then
    say "$dir: would run $TIDY_ENV go mod tidy"
    return 0
  fi
  if [[ -d "$dir" ]]; then
    pushd "$dir" >/dev/null
    run "$TIDY_ENV go mod tidy"
    popd >/dev/null
  fi
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

# Partition the bump set so foundation modules (x/rkit, x/p, …) are tagged
# before kit and other dependents that now require them.
FOUNDATION=()
CORE=()
REST=()
for dir in "${DIRS[@]}"; do
  if is_foundation_dir "$dir"; then
    FOUNDATION+=("$dir")
  elif is_core_dir "$dir"; then
    CORE+=("$dir")
  else
    REST+=("$dir")
  fi
done

say "Foundation (no workspace deps): ${FOUNDATION[*]:-(none)}"
say "Core (foundation deps only, e.g. kit -> x/rkit, x/p): ${CORE[*]:-(none)}"
say "Remaining dependents: ${REST[*]:-(none)}"

# Tag foundation modules at the current commit first and push so kit (and
# other core modules) can resolve the new x/rkit / x/p versions from git.
if [[ ${#FOUNDATION[@]} -gt 0 ]]; then
  say "Tagging foundation modules first"
  for dir in "${FOUNDATION[@]}"; do
    i=$(idx_by_dir "$dir")
    create_tag "$dir" "${NEXTS[$i]}"
  done
  if [[ $DRY_RUN -eq 1 ]]; then
    say "Would push foundation tags"
  else
    run "git push --tags"
  fi
fi

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

    # Find the index that provides this dep_path (bump set only: we only
    # rewrite requires for modules that are also receiving a new tag).
    dep_idx=$(idx_by_path "$dep_path")
    if [[ "$dep_idx" == "-1" ]]; then
      continue
    fi

    new_ver="${NEXTS[$dep_idx]}"
    # Determine current required version (if any) by parsing this module's go.mod
    curr_req=$(awk -v m="$dep_path" 'inb && $1==m {print $2} /^require \(/ {inb=1} /^\)/{inb=0} /^require / {if ($2==m) print $3}' "$gomod" | sed -E 's/[\"]//g' | head -n1 || true)

    if [[ -n "$new_ver" ]]; then
      append_unique changed_modules "$dir"
      if [[ $DRY_RUN -eq 1 ]]; then
        say "$dir: would set require $dep_path@$new_ver (was ${curr_req:-unset})"
      else
        pushd "$dir" >/dev/null
        go mod edit -require="$dep_path@$new_ver"
        popd >/dev/null
        say "$dir: set require $dep_path@$new_ver (was ${curr_req:-unset})"
      fi
    fi
  done < "$__tmp_deps"
  rm -f "$__tmp_deps"
done

# Tidy core modules now that foundation tags are on the remote. This writes
# kit/go.sum hashes for the new x/rkit and x/p versions into the same commit
# that will be tagged as the new kit version.
if [[ ${#CORE[@]} -gt 0 ]]; then
  for dir in "${CORE[@]}"; do
    tidy_dir "$dir"
    append_unique changed_modules "$dir"
  done
fi

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

# Tag core + remaining dependents at the require-update commit
for dir in "${CORE[@]:-}" "${REST[@]:-}"; do
  [[ -z "$dir" ]] && continue
  i=$(idx_by_dir "$dir")
  [[ "$i" == "-1" ]] && continue
  create_tag "$dir" "${NEXTS[$i]}"
done

if [[ $DRY_RUN -eq 1 ]]; then
  say "Would push remaining tags"
else
  run "git push --tags"
fi

# Tidy remaining dependents now that kit (and other core) tags exist.
if [[ ${#REST[@]} -gt 0 ]]; then
  if [[ $DRY_RUN -eq 1 ]]; then
    for d in "${REST[@]}"; do
      say "$d: would run $TIDY_ENV go mod tidy (post-tag)"
    done
    say "Would commit post-tidy changes and push"
  else
    for d in "${REST[@]}"; do
      tidy_dir "$d"
    done
    post_msg="bump workspace post-tidy ($PART)"
    for d in "${REST[@]}"; do
      run "git add $d/go.mod || true"
      run "git add $d/go.sum || true"
    done
    run "git commit -m '$post_msg' || true"
    run "git push || true"
  fi
fi

say "Done."
