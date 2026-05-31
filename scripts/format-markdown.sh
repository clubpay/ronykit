#!/usr/bin/env bash
# format-markdown.sh — format Markdown like make format-md, but preserve YAML frontmatter.
#
# markdownfmt treats leading "---" as a horizontal rule and corrupts frontmatter
# (e.g. merges "applies_to_files:" onto the previous list item). Files that start
# with "---\n" are split: frontmatter is kept byte-for-byte, only the body is
# passed to markdownfmt.

set -euo pipefail

if ! command -v markdownfmt >/dev/null 2>&1; then
	echo "markdownfmt not found; run: make setup" >&2
	exit 1
fi

format_file() {
	local file=$1

	if [[ ! -f $file ]]; then
		return 0
	fi

	# No YAML frontmatter — format the whole file in place.
	if [[ $(head -c 4 "$file") != ---$'\n' ]]; then
		markdownfmt -w -gofmt "$file"
		return 0
	fi

	local rest yaml_block body tmp_body tmp_out
	rest=$(tail -c +5 "$file") # strip leading "---\n"

	if [[ $rest != *$'\n---\n'* ]]; then
		markdownfmt -w -gofmt "$file"
		return 0
	fi

	yaml_block=${rest%%$'\n---\n'*}
	body=${rest#*$'\n---\n'}

	tmp_body=$(mktemp)
	tmp_out=$(mktemp)
	trap 'rm -f "$tmp_body" "$tmp_out"' RETURN

	printf '%s' "$body" >"$tmp_body"
	markdownfmt -gofmt "$tmp_body" >"$tmp_out"

	{
		printf '%s\n' '---'
		printf '%s' "$yaml_block"
		[[ $yaml_block == *$'\n' ]] || printf '\n'
		printf '%s\n' '---'
		cat "$tmp_out"
	} >"$file"
}

export -f format_file

find . -not -path './.git/*' -type f \( -iname '*.md' \) -print0 |
	xargs -0 -n1 bash -c 'format_file "$0"' bash
