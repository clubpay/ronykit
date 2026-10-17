---
import_path: github.com/clubpay/ronykit/x/rkit
short_name: rkit
---

Shared, dependency-light helpers used across every RonyKIT service: random IDs, JSON/byte/string casts, string↔number conversions, case transforms, generic collection operations, pointer/optional helpers, struct mapping, time, and file utilities. Reach here FIRST instead of hand-writing utilities or pulling stdlib/third-party equivalents — `rkit` keeps services consistent and is the single place these helpers are tuned (e.g. zero-copy casts, pooled RNG).

## Usage Hint

Import as `github.com/clubpay/ronykit/x/rkit`.

### IDs & random (`random.go`)

- `rkit.RandomID(n)` — alphanumeric ID of length n. `rkit.RandomIDs(n...)` — batch.
- `rkit.RandomDigit(n)` — digits-only string (OTPs, numeric codes).
- `rkit.RandomInt`/`RandomInt32`/`RandomInt64`/`RandomUint64` — fast bounded pseudo-random ints.
- `rkit.SecureRandomInt63`/`SecureRandomUint64` — crypto-grade randomness when security matters (`RandomID` is fast but not secure).
- `rkit.FastRand()` — the runtime's per-thread fast random; prefer the helpers above.
- Do NOT use `github.com/google/uuid`, `crypto/rand`, or `math/rand` directly.

### JSON & casts (`cast.go`, `unsafe.go`)

- `rkit.ToJSON(v)` / `rkit.ToJSONStr(v)` / `rkit.FromJSON[T](b)` — error-free (de)serialization for internal use.
- `rkit.CastJSON[T](v)` — convert between types via their JSON form. `rkit.ToMap(v)` — to `map[string]any`.
- `rkit.Cast[T](any)` — safe type assertion (zero value on miss). `rkit.TryCast[T](any)`.
- `rkit.B2S(b)` / `rkit.S2B(s)` (aliases of `ByteToStr` / `StrToByte`) — zero-copy `[]byte`↔`string`; the result aliases the input, so never mutate either side while both are in use. `rkit.CloneStr`/`CloneBytes` for owned copies.
- `rkit.Numeric` / `rkit.ParseNumeric(v)` — a number decodable from JSON string, int or float.

### String ↔ number (`str.go`)

- `rkit.StrToInt`/`StrToInt32`/`StrToInt64`/`StrToUInt`/`StrToUInt32`/`StrToUInt64`/`StrToFloat32`/`StrToFloat64` — ignore the parse error: malformed → 0, out-of-range integers clamp to the type's bounds (floats → ±Inf), negative → 0 for unsigned. Use `strconv` when you need the error.
- `rkit.IntToStr`/`Int32ToStr`/`Int64ToStr`/`UIntToStr`/`UInt32ToStr`/`UInt64ToStr`/`Float32ToStr`/`Float64ToStr` (aliases `F32ToStr`/`F64ToStr`).
- `rkit.StrTruncate(s, n)` (by rune, no ellipsis). Use `rkit.L(format, args...)` (a `StrLine`) + `rkit.StrLines(...)` to build multi-line output.
- Prefer these over raw `strconv` when an error-free conversion is what you want.

### Case transforms (`transform.go`)

- `rkit.ToCamel` / `ToLowerCamel` / `ToSnake` / `ToSnakeWithIgnore` / `ToScreamingSnake` / `ToKebab` / `ToScreamingKebab` / `ToDelimited` / `ToScreamingDelimited`.

### Collections (generics) (`array.go`, `generic.go`)

- `rkit.Map` / `rkit.Filter` / `rkit.Reduce` / `rkit.Paginate` / `rkit.ForEach`. `Map`/`Filter` always return non-nil slices; `Reduce(fn, s)` takes the function first.
- `rkit.Contains` / `ContainsAny` / `ContainsAll` / `AddUnique`.
- `rkit.ArrayToMap` / `ArrayToMapFunc` / `ArrayToSet` (`map[T]struct{}`) / `MapToArray` / `MapToArrayFunc` / `MapKeysToArray` / `TransformMap`.
- `rkit.First` / `FirstOr` — first present value among keys.

### Pointers, zero values, error-handling (`generic.go`)

- `rkit.PtrVal(*T)` / `rkit.ValPtrOrNil(T)` for pointer/optional fields. For a plain pointer use the `new(v)` builtin (Go 1.26+); `rkit.ValPtr(T)` is kept for older code and `go fix` inlines it.
- `rkit.Coalesce(...T)` — first non-zero value (evaluates every argument first).
- `rkit.Must(v, err)` / `rkit.Assert(err)` for unrecoverable startup preconditions; `rkit.Ok(v, _)` / `rkit.OkOr(v, err, fallback)` to ignore/replace errors.

### Struct mapping (`cast.go`)

- `rkit.DynCast[DST](src, mappingPair...)` / `rkit.DynCastOption[DST](src, converters, mappingPair...)` for deep struct-to-struct copies; build each `TypeConverter` with `rkit.TypeConvert`. Use instead of calling `jinzhu/copier` directly.

### Time & files (`time.go`, `file.go`)

- `rkit.TimeUnix()` (cached per-second clock, up to 1s stale), `rkit.TimeUnixAdd`/`TimeUnixSubtract`, `rkit.NanoTime`/`CPUTicks` for cheap durations.
- `rkit.GetExecDir`/`GetExecName`/`GetCurrentDir`, `rkit.CopyFile`/`CopyFileWithBuffer`, `rkit.ReadYamlFile`/`WriteYamlFile`.
- `rkit.SpinLock` (`spinlock.go`) only for exceptionally short critical sections; use `sync.Mutex` otherwise.

### Struct field access (`reflector`)

- Import as `github.com/clubpay/ronykit/x/rkit/reflector`.
- `reflector.New()` / `reflector.Register(m, tags...)` / `Reflector.Load` for cached struct field lookup by name or tag: `Load` returns a `Reflected` (`Obj()`, `ByTag`) whose `Fields` map holds one `FieldInfo` per field.
- Errors: `reflector.ErrNoField`, `reflector.ErrNotExported`, `reflector.ErrMessageIsNotStruct`.

### Enforcement

`ronyup lint` (MCP tool `check_rkit_usage`) reports hand-written copies of these helpers — if-err-panic, loops that only
collect map keys/values, unsafe `[]byte`↔`string` casts, local `must`/`deref`/`contains`/`coalesce` — and `make verify`
runs it on every feature module. The monorepo's full task-organised reference is `x/rkit/README.md`.
