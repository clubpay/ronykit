#!/usr/bin/env bash

# render-homebrew-formula.sh — Render the Homebrew formula from template + checksums
#
# Usage:
#   scripts/render-homebrew-formula.sh \
#     --tag ronyup/v0.4.7 \
#     --repo clubpay/ronykit \
#     --checksums /path/to/checksums.env \
#     --output /path/to/ronyup.rb
#
# checksums.env must define:
#   SHA256_DARWIN_AMD64, SHA256_DARWIN_ARM64,
#   SHA256_LINUX_AMD64, SHA256_LINUX_ARM64

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

TAG=""
REPO="clubpay/ronykit"
CHECKSUMS=""
OUTPUT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag)
      TAG="${2:-}"
      shift 2
      ;;
    --repo)
      REPO="${2:-}"
      shift 2
      ;;
    --checksums)
      CHECKSUMS="${2:-}"
      shift 2
      ;;
    --output)
      OUTPUT="${2:-}"
      shift 2
      ;;
    -h|--help)
      echo "Usage: $0 --tag <tag> --checksums <file> --output <file> [--repo owner/name]"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "$TAG" || -z "$CHECKSUMS" || -z "$OUTPUT" ]]; then
  echo "--tag, --checksums, and --output are required" >&2
  exit 1
fi

if [[ ! -f "$CHECKSUMS" ]]; then
  echo "Checksums file not found: $CHECKSUMS" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$CHECKSUMS"

for key in SHA256_DARWIN_AMD64 SHA256_DARWIN_ARM64 SHA256_LINUX_AMD64 SHA256_LINUX_ARM64; do
  if [[ -z "${!key:-}" ]]; then
    echo "Missing $key in $CHECKSUMS" >&2
    exit 1
  fi
done

VERSION="${TAG##*/v}"
TAG_ENC="$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=""))' "$TAG")"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG_ENC}"

asset_url() {
  local platform="$1"
  echo "${BASE_URL}/ronyup_${VERSION}_${platform}.tar.gz"
}

TEMPLATE="$ROOT_DIR/.github/homebrew/ronyup.rb.tmpl"
if [[ ! -f "$TEMPLATE" ]]; then
  echo "Template not found: $TEMPLATE" >&2
  exit 1
fi

sed \
  -e "s|{{VERSION}}|${VERSION}|g" \
  -e "s|{{URL_DARWIN_AMD64}}|$(asset_url darwin_amd64)|g" \
  -e "s|{{URL_DARWIN_ARM64}}|$(asset_url darwin_arm64)|g" \
  -e "s|{{URL_LINUX_AMD64}}|$(asset_url linux_amd64)|g" \
  -e "s|{{URL_LINUX_ARM64}}|$(asset_url linux_arm64)|g" \
  -e "s|{{SHA256_DARWIN_AMD64}}|${SHA256_DARWIN_AMD64}|g" \
  -e "s|{{SHA256_DARWIN_ARM64}}|${SHA256_DARWIN_ARM64}|g" \
  -e "s|{{SHA256_LINUX_AMD64}}|${SHA256_LINUX_AMD64}|g" \
  -e "s|{{SHA256_LINUX_ARM64}}|${SHA256_LINUX_ARM64}|g" \
  "$TEMPLATE" > "$OUTPUT"
