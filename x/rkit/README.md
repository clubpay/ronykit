# rkit

The `rkit` package provides small shared helpers for RonyKIT modules and scaffolded services. Helpers cover IDs, JSON, zero-copy casts, string and number conversions, case transforms, collections, pointers, errors, struct mapping, time, and files.

```go
import "github.com/clubpay/ronykit/x/rkit"
```

**Check rkit before you write a helper.** In modules that require `x/rkit`, `make lint-rkit` runs `ronyup lint`. This command reports manual copies of rkit helpers.

Reported patterns include error panics, default-value conditions, map and filter loops, map key and value loops, dropped errors, dropped type assertions, and unsafe casts. The linter also covers standard-library calls such as `slices.Contains`, `cmp.Or`, and `strconv.Itoa`.

The `depguard` linter rejects third-party substitutes, including `samber/lo`, `strcase`, uuid, ulid, xid, and `jinzhu/copier`.

Use rkit when rkit and the standard library cover the same case. `ValPtr` is the exception. In Go 1.26 and later, `go fix` replaces it with the `new(v)` builtin.

## Select a helper by task

### IDs and randomness

| Requirement | Use | Notes |
|-------------|-----|-------|
| Alphanumeric ID or token | `RandomID(n)`, `RandomIDs(n...)` | Fast and not cryptographically secure |
| Numeric code such as OTP or PIN | `RandomDigit(n)` | Uses the same generator as `RandomID` |
| Bounded pseudo-random number | `RandomInt`, `RandomInt32`, `RandomInt64`, `RandomUint64` | `n == 0` means no bound |
| Secure randomness | `SecureRandomUint64()`, `SecureRandomInt63(n)` | Uses `crypto/rand` |
| Runtime per-thread fast random value | `FastRand()` | Linked to the runtime. Prefer the other helpers. |

### JSON, casts, and zero-copy conversions

| Requirement | Use | Notes |
|-------------|-----|-------|
| Marshal without error handling | `ToJSON(v)`, `ToJSONStr(v)` | Returns empty output on error. Use for values that always encode. |
| Unmarshal into a type | `FromJSON[T](b)` | Returns the zero value on error |
| Convert types through their JSON form | `CastJSON[T](v)`, `ToMap(v)` | Partially fills the result on mismatch |
| Type assertion with zero-value fallback | `Cast[T](v)`, `TryCast[T](v)` | Equivalent helpers |
| Convert `[]byte` to `string` without a copy | `B2S(b)` or alias `ByteToStr` | The string aliases `b`. Do not modify `b` while the string is in use. |
| Convert `string` to `[]byte` without a copy | `S2B(s)` or alias `StrToByte` | Never write to the result. Returns `nil` for `""`. |
| Owned copy | `CloneStr(s)`, `CloneBytes(b)` | |
| Number decoded from string, int, or float | `Numeric`, `ParseNumeric(v)` | Marshals as a string with fixed precision |

### Strings and numbers

| Requirement | Use | Notes |
|-------------|-----|-------|
| Parse and ignore errors | `StrToInt`, `StrToInt32`, `StrToInt64`, `StrToUInt`, `StrToUInt32`, `StrToUInt64`, `StrToFloat32`, `StrToFloat64` | Malformed input returns 0. Out-of-range integers clamp to the type bounds. Out-of-range floats become ±Inf. Negative unsigned values return 0. Use `strconv` when you need the error. |
| Format | `IntToStr`, `Int32ToStr`, `Int64ToStr`, `UIntToStr`, `UInt32ToStr`, `UInt64ToStr`, `Float32ToStr`, `Float64ToStr`, `F32ToStr`, `F64ToStr` | Floats use the shortest round-trip form without an exponent |
| Truncate to n runes | `StrTruncate(s, n)` | Cuts by rune and adds no ellipsis |
| Multi-line output | `L(format, args...)`, `StrLine`, `StrLines(lines...)` | Runs `fmt.Sprintf` once per line and joins with `\n` |

### Case transforms

| Requirement | Use |
|-------------|-----|
| `CamelCase` or `camelCase` | `ToCamel`, `ToLowerCamel` |
| `snake_case` or `SCREAMING_SNAKE` | `ToSnake`, `ToScreamingSnake`, `ToSnakeWithIgnore` |
| `kebab-case` or `SCREAMING-KEBAB` | `ToKebab`, `ToScreamingKebab` |
| Any delimiter | `ToDelimited(s, '.')`, `ToScreamingDelimited(s, '.', ignore, screaming)` |

Words split at case changes and digit boundaries. For example, `JSONData2` becomes `json_data_2`.

### Slices and maps

| Requirement | Use | Notes |
|-------------|-----|-------|
| Transform each element | `Map(s, fn)` | Always returns a non-nil slice. JSON output is `[]`, not `null`. |
| Keep matching elements | `Filter(s, fn)` | Returns a non-nil slice with clipped capacity |
| Fold to one value | `Reduce(fn, s)` | Starts from the zero value. The function comes first. |
| Mutate in place | `ForEach(s, func(*T))` | |
| Process in pages | `Paginate(s, size, func(start, end int) error)` | Calls `fn(0, 0)` once for an empty slice |
| Test membership | `Contains`, `ContainsAny`, `ContainsAll` | `Contains` matches `slices.Contains`. `ContainsAll(s, nil)` is true. |
| Append only if absent | `AddUnique(s, v)` | Uses a linear scan |
| Convert map keys or values to a slice | `MapKeysToArray(m)`, `MapToArray(m)`, `MapToArrayFunc(m, fn)` | Results are unordered. Sort afterward if order matters. |
| Convert a slice to a map | `ArrayToMap(s)`, `ArrayToMapFunc(s, keyFn)` | `ArrayToMap` uses the slice index |
| Convert a slice to a set | `ArrayToSet(s)` | Returns `map[T]struct{}`. Look up with `_, ok := set[x]`. |
| Rewrite a map | `TransformMap(m, func(k, v) (k2, v2))` | |
| First present key | `First(m, keys...)`, `FirstOr(def, m, keys...)` | |

### Pointers, defaults, and errors

| Requirement | Use | Notes |
|-------------|-----|-------|
| Nil-safe dereference | `PtrVal(p)` | Returns the zero value for `nil` |
| Pointer to a value | `new(v)` in Go 1.26 and later | `ValPtr(v)` remains for older code. `go fix` inlines it. |
| Pointer or `nil` for the zero value | `ValPtrOrNil(v)` | |
| First non-zero value | `Coalesce(a, b, c)` | Evaluates every argument first. Do not pass expensive calls. |
| Panic on error during startup or in `MustX` wrappers | `Must(f())`, `Assert(err)` | |
| Drop or replace an error | `Ok(f())`, `OkOr(v, err, fallback)` | |

### Struct mapping

| Requirement | Use | Notes |
|-------------|-----|-------|
| Deep struct-to-struct copy | `DynCast[DST](src, "SrcField", "DstField", ...)` | Panics on error |
| Copy with custom conversions | `DynCastOption[DST](src, converters, ...)` with `TypeConvert(fn)` and `TypeConverter` | Use instead of `jinzhu/copier` |

### Time, files, and locking

| Requirement | Use | Notes |
|-------------|-----|-------|
| Current Unix seconds on a hot path | `TimeUnix()` | Can be up to one second old. Starts a refresh goroutine on the first call. |
| Offset a Unix timestamp | `TimeUnixAdd(t, d)`, `TimeUnixSubtract(t, d)` | Second resolution |
| Low-cost monotonic durations | `NanoTime()`, `CPUTicks()` | Runtime-linked and not wall-clock time |
| Executable or working directory | `GetExecDir()`, `GetExecName()`, `GetCurrentDir()` | `GetExecDir` and `GetCurrentDir` return `""` on error |
| Copy a file | `CopyFile(src, dst)`, `CopyFileWithBuffer(src, dst, buf)` | Supports OS paths, not `fs.FS` |
| YAML file I/O | `ReadYamlFile(path, &v)`, `WriteYamlFile(path, v)` | |
| Lock for very short critical sections | `SpinLock` | Spins with `runtime.Gosched`. Use `sync.Mutex` otherwise. |

### Cached struct field access with `rkit/reflector`

```go
import "github.com/clubpay/ronykit/x/rkit/reflector"
```

`reflector.Register(m, tags...)` caches a struct type's `Fields`. It creates one `FieldInfo` for each field, indexed by name or tag. Later lookups do not repeat reflection.

`reflector.New()` returns a `Reflector`. `Load(m, tags...)` returns the `Reflected` view with `Obj()` and `ByTag(tag)`. Use `Get`, `GetString`, and `GetInt` to read one field.

Possible errors are `ErrNoField`, `ErrNotExported`, and `ErrMessageIsNotStruct`.

## Keep this document current

Every exported identifier must appear in this README and `ronyup/cmd/mcp/knowledge/resources/packages/rkit.md`. A ronyup test checks this requirement.

When you add a helper, update both documents. Also decide whether `ronyup lint` in `ronyup/internal/rkitlint` must detect manual copies.
