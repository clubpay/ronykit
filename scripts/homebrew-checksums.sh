#!/usr/bin/env bash

# homebrew-checksums.sh — Compute Homebrew formula checksums for ronyup tarballs
#
# From local build artifacts:
#   scripts/homebrew-checksums.sh --from-dir dist --version 0.4.7 --output checksums.env
#
# From an existing GitHub release:
#   scripts/homebrew-checksums.sh --tag ronyup/v0.4.7 --repo clubpay/ronykit --output checksums.env

set -euo pipefail

FROM_DIR=""
VERSION=""
TAG=""
REPO="clubpay/ronykit"
OUTPUT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --from-dir)
      FROM_DIR="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --tag)
      TAG="${2:-}"
      shift 2
      ;;
    --repo)
      REPO="${2:-}"
      shift 2
      ;;
    --output)
      OUTPUT="${2:-}"
      shift 2
      ;;
    -h|--help)
      echo "Usage: $0 (--from-dir <dir> --version <ver> | --tag <tag> [--repo owner/name]) --output <file>"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "$OUTPUT" ]]; then
  echo "--output is required" >&2
  exit 1
fi

if [[ -n "$FROM_DIR" && -n "$TAG" ]]; then
  echo "Use either --from-dir or --tag, not both" >&2
  exit 1
fi

if [[ -n "$FROM_DIR" && -z "$VERSION" ]]; then
  echo "--version is required with --from-dir" >&2
  exit 1
fi

if [[ -z "$FROM_DIR" && -z "$TAG" ]]; then
  echo "Either --from-dir or --tag is required" >&2
  exit 1
fi

if [[ -n "$TAG" && -z "$VERSION" ]]; then
  VERSION="${TAG##*/v}"
fi

: > "$OUTPUT"

for platform in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
  upper="$(echo "$platform" | tr '[:lower:]' '[:upper:]' | tr '-' '_')"
  key="SHA256_${upper}"
  file="ronyup_${VERSION}_${platform}.tar.gz"

  if [[ -n "$FROM_DIR" ]]; then
    path="${FROM_DIR}/${file}"
    if [[ ! -f "$path" ]]; then
      echo "Missing build artifact: $path" >&2
      exit 1
    fi
    sha="$(sha256sum "$path" | cut -d' ' -f1)"
  else
    TAG_ENC="$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=""))' "$TAG")"
    url="https://github.com/${REPO}/releases/download/${TAG_ENC}/${file}"
    tmp="$(mktemp)"
    if ! curl -fsSL "$url" -o "$tmp"; then
      echo "Failed to download ${url}" >&2
      echo "No GitHub release assets found for tag '${TAG}'." >&2
      echo "Run 'make ronyup-release TAG=${TAG}' first to build and publish binaries." >&2
      rm -f "$tmp"
      exit 1
    fi
    sha="$(sha256sum "$tmp" | cut -d' ' -f1)"
    rm -f "$tmp"
  fi

  echo "${key}=${sha}" >> "$OUTPUT"
  echo "${key}=${sha}"
done
