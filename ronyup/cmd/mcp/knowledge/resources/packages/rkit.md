---
import_path: github.com/clubpay/ronykit/x/rkit
short_name: rkit
---

`rkit` provides shared, dependency-light helpers for RonyKIT services. Helpers cover random IDs, JSON/byte/string casts, string and number conversions, case transforms, generic collections, pointers, optional values, struct mapping, time, and files.

Check rkit before writing utilities or importing standard-library or third-party equivalents. Central helpers keep services consistent and provide optimized implementations, including zero-copy casts and pooled random generation.

## Usage Hint

Import `github.com/clubpay/ronykit/x/rkit`.

### IDs and random values in `random.go`

- `rkit.RandomID(n)` creates an alphanumeric ID of length n. `rkit.RandomIDs(n...)` creates a batch.
- `rkit.RandomDigit(n)` creates a digits-only string for OTPs and numeric codes.
- `rkit.RandomInt`, `RandomInt32`, `RandomInt64`, and `RandomUint64` create fast bounded pseudo-random integers.
- `rkit.SecureRandomInt63` and `SecureRandomUint64` provide cryptographic randomness. `RandomID` is fast but not secure.
- `rkit.FastRand()` uses the runtime's per-thread fast random generator. Prefer the other helpers.
- Do not use `github.com/google/uuid`, `crypto/rand`, or `math/rand` directly.

### JSON and casts in `cast.go` and `unsafe.go`

- Use `rkit.ToJSON(v)`, `rkit.ToJSONStr(v)`, and `rkit.FromJSON[T](b)` for error-free internal serialization and deserialization.
- `rkit.CastJSON[T](v)` converts types through their JSON form. `rkit.ToMap(v)` converts to `map[string]any`.
- `rkit.Cast[T](any)` returns the zero value on a failed type assertion. `rkit.TryCast[T](any)` provides the same behavior.
- `rkit.B2S(b)` and `rkit.S2B(s)` perform zero-copy `[]byte` and `string` conversions. Their aliases are `ByteToStr` and `StrToByte`. The result aliases the input. Never mutate either side while both are in use. Use `rkit.CloneStr` and `CloneBytes` for owned copies.
- `rkit.Numeric` and `rkit.ParseNumeric(v)` decode a number from a JSON string, int, or float.

### String and number conversions in `str.go`

- Use `rkit.StrToInt`, `StrToInt32`, `StrToInt64`, `StrToUInt`, `StrToUInt32`, `StrToUInt64`, `StrToFloat32`, and `StrToFloat64` to ignore parse errors. Malformed input returns 0. Out-of-range integers clamp to the type bounds. Out-of-range floats become ±Inf. Negative unsigned values return 0. Use `strconv` when you need the error.
- Format with `rkit.IntToStr`, `Int32ToStr`, `Int64ToStr`, `UIntToStr`, `UInt32ToStr`, `UInt64ToStr`, `Float32ToStr`, and `Float64ToStr`. Aliases are `F32ToStr` and `F64ToStr`.
- `rkit.StrTruncate(s, n)` truncates by rune without an ellipsis. Use `rkit.L(format, args...)`, `StrLine`, and `rkit.StrLines(...)` for multi-line output.
- Prefer these helpers over raw `strconv` for error-free conversion.

### Case transforms in `transform.go`

- Use `rkit.ToCamel`, `ToLowerCamel`, `ToSnake`, `ToSnakeWithIgnore`, `ToScreamingSnake`, `ToKebab`, `ToScreamingKebab`, `ToDelimited`, and `ToScreamingDelimited`.

### Generic collections in `array.go` and `generic.go`

- Use `rkit.Map`, `Filter`, `Reduce`, `Paginate`, and `ForEach`. `Map` and `Filter` always return non-nil slices. `Reduce(fn, s)` takes the function first.
- Use `rkit.Contains`, `ContainsAny`, `ContainsAll`, and `AddUnique` for membership.
- Use `rkit.ArrayToMap`, `ArrayToMapFunc`, `ArrayToSet`, `MapToArray`, `MapToArrayFunc`, `MapKeysToArray`, and `TransformMap` for collection conversion. `ArrayToSet` returns `map[T]struct{}`.
- `rkit.First` and `FirstOr` return the first present value among keys.

### Pointers, zero values, and errors in `generic.go`

- Use `rkit.PtrVal(*T)` and `rkit.ValPtrOrNil(T)` for pointer and optional fields. For a plain pointer, use the `new(v)` builtin in Go 1.26 and later. `rkit.ValPtr(T)` remains for older code. `go fix` inlines it.
- `rkit.Coalesce(...T)` returns the first non-zero value. It evaluates every argument first.
- Use `rkit.Must(v, err)` and `rkit.Assert(err)` for unrecoverable startup preconditions. Use `rkit.Ok(v, _)` and `rkit.OkOr(v, err, fallback)` to ignore or replace errors.

### Struct mapping in `cast.go`

- Use `rkit.DynCast[DST](src, mappingPair...)` and `rkit.DynCastOption[DST](src, converters, mappingPair...)` for deep struct copies. Build each `TypeConverter` with `rkit.TypeConvert`. Use these helpers instead of `jinzhu/copier`.

### Time and files in `time.go` and `file.go`

- `rkit.TimeUnix()` returns a cached per-second clock that can be up to one second old. Use `rkit.TimeUnixAdd` and `TimeUnixSubtract` for offsets. Use `rkit.NanoTime` and `CPUTicks` for low-cost durations.
- Use `rkit.GetExecDir`, `GetExecName`, `GetCurrentDir`, `CopyFile`, `CopyFileWithBuffer`, `ReadYamlFile`, and `WriteYamlFile`.
- Use `rkit.SpinLock` from `spinlock.go` only for exceptionally short critical sections. Use `sync.Mutex` otherwise.

### Struct field access with `reflector`

- Import `github.com/clubpay/ronykit/x/rkit/reflector`.
- Use `reflector.New()`, `reflector.Register(m, tags...)`, and `Reflector.Load` for cached struct field lookup by name or tag. `Load` returns `Reflected` with `Obj()` and `ByTag`. Its `Fields` map contains one `FieldInfo` for each field.
- Errors are `reflector.ErrNoField`, `reflector.ErrNotExported`, and `reflector.ErrMessageIsNotStruct`.

### Enforcement

`ronyup lint` and MCP tool `check_rkit_usage` report manual copies of these helpers. Patterns include error panics, default-value conditions, and map or filter loops. They also include map key and value loops, dropped errors, dropped type assertions, unsafe casts, and covered standard-library calls.

`make verify` runs the check on every feature module. Use rkit when rkit and the standard library overlap. See `x/rkit/README.md` for the full task-oriented reference.
