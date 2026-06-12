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
- `rkit.SecureRandomInt63`/`SecureRandomUint64` — crypto-grade randomness when security matters.
- Do NOT use `github.com/google/uuid`, `crypto/rand`, or `math/rand` directly.

### JSON & casts (`cast.go`, `unsafe.go`)

- `rkit.ToJSON(v)` / `rkit.ToJSONStr(v)` / `rkit.FromJSON[T](b)` — error-free (de)serialization for internal use.
- `rkit.CastJSON[T](v)` — convert between types via their JSON form. `rkit.ToMap(v)` — to `map[string]any`.
- `rkit.Cast[T](any)` — safe type assertion (zero value on miss). `rkit.TryCast[T](any)`.
- `rkit.B2S(b)` / `rkit.S2B(s)` — zero-copy `[]byte`↔`string`. `rkit.CloneStr`/`CloneBytes` for owned copies.

### String ↔ number (`str.go`)

- `rkit.StrToInt`/`StrToInt32`/`StrToInt64`/`StrToUInt*`/`StrToFloat32`/`StrToFloat64`.
- `rkit.IntToStr`/`Int32ToStr`/`Int64ToStr`/`UInt*ToStr`/`Float32ToStr`/`Float64ToStr` (aliases `F32ToStr`/`F64ToStr`).
- `rkit.StrTruncate(s, n)`. Use `rkit.L(format, args...)` + `rkit.StrLines(...)` to build multi-line output.
- Prefer these over raw `strconv`.

### Case transforms (`transform.go`)

- `rkit.ToCamel` / `ToLowerCamel` / `ToSnake` / `ToScreamingSnake` / `ToKebab` / `ToScreamingKebab` / `ToDelimited`.

### Collections (generics) (`array.go`, `generic.go`)

- `rkit.Map` / `rkit.Filter` / `rkit.Reduce` / `rkit.Paginate` / `rkit.ForEach`.
- `rkit.Contains` / `ContainsAny` / `ContainsAll` / `AddUnique`.
- `rkit.ArrayToMap` / `ArrayToMapFunc` / `ArrayToSet` / `MapToArray` / `MapToArrayFunc` / `MapKeysToArray` / `TransformMap`.
- `rkit.First` / `FirstOr` — first present value among keys.

### Pointers, zero values, error-handling (`generic.go`)

- `rkit.PtrVal(*T)` / `rkit.ValPtr(T)` / `rkit.ValPtrOrNil(T)` for pointer/optional fields.
- `rkit.Coalesce(...T)` — first non-zero value.
- `rkit.Must(v, err)` / `rkit.Assert(err)` for unrecoverable startup preconditions; `rkit.Ok(v, _)` / `rkit.OkOr(v, err, fallback)` to ignore/replace errors.

### Struct mapping (`cast.go`)

- `rkit.DynCast[DST](src, mappingPair...)` / `rkit.DynCastOption[DST](src, converters, mappingPair...)` for deep struct-to-struct copies; register converters with `rkit.TypeConvert`. Use instead of calling `jinzhu/copier` directly.

### Time & files (`time.go`, `file.go`)

- `rkit.TimeUnix()` (cached per-second clock), `rkit.NanoTime`/`CPUTicks` for cheap durations.
- `rkit.GetExecDir`/`GetExecName`/`GetCurrentDir`, `rkit.CopyFile`, `rkit.ReadYamlFile`/`WriteYamlFile`.
- `rkit.SpinLock` (`spinlock.go`) for very short critical sections instead of `sync.Mutex`.
