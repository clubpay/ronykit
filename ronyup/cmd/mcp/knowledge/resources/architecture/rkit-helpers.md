Prefer `x/rkit` helpers over reinventing common utilities:

- `rkit.RandomID(n)` / `rkit.RandomIDs(...n)` for IDs and short tokens.
- `rkit.ToJSON` / `rkit.FromJSON` for byte-safe serialization (and `rkit.B2S` /
  `rkit.S2B` for zero-copy `[]byte`/`string` conversions).
- `rkit.Map` / `rkit.Filter` / `rkit.Reduce` for generic collection transforms.
- `rkit.ToCamel` / `rkit.ToSnake` / `rkit.ToKebab` for case transforms.
- `rkit.Coalesce` for first-non-zero selection.
- `rkit.L` / `rkit.StrLines` for structured multi-line string building (used by
  the MCP tool result helpers).
- `rkit.Must` / `rkit.Assert` for unrecoverable preconditions in startup code.
