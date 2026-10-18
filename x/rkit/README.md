# rkit

Small, shared helpers used across RonyKIT modules and scaffolded services: IDs, JSON and zero-copy casts, string↔number
conversions, case transforms, collection helpers, pointer/error helpers, struct mapping, time and files.

```go
import "github.com/clubpay/ronykit/x/rkit"
```

**Reach here before writing a helper.** In modules that require `x/rkit`, `make lint-rkit` (`ronyup lint`) flags
code that re-implements them (if-err-panic, default-value ifs, map/filter loops, map key/value loops, dropped errors and type-assertion results, unsafe casts, standard-library calls rkit covers (`slices.Contains`, `cmp.Or`, `strconv.Itoa`, ...), local `must`/`deref`/`contains`/`coalesce`), and depguard rejects third-party substitutes (`samber/lo`, `strcase`, uuid/ulid/xid,
`jinzhu/copier`). When rkit and the standard library both cover a case, use rkit; the exception is `ValPtr`, which
`go fix` inlines to the `new(v)` builtin (Go 1.26+).

## Pick a helper by task

### IDs and randomness

| Need                                  | Use                                                  | Notes                                       |
|---------------------------------------|------------------------------------------------------|---------------------------------------------|
| An alphanumeric ID or token           | `RandomID(n)`, `RandomIDs(n...)`                     | Fast, **not** cryptographically secure      |
| A numeric code (OTP, PIN)             | `RandomDigit(n)`                                     | Same generator as `RandomID`                |
| A bounded pseudo-random number        | `RandomInt`, `RandomInt32`, `RandomInt64`, `RandomUint64` | `n == 0` means no bound                 |
| Secure randomness                     | `SecureRandomUint64()`, `SecureRandomInt63(n)`       | Backed by `crypto/rand`                     |
| The runtime's per-thread fast random  | `FastRand()`                                         | Linked to the runtime; prefer the above     |

### JSON, casts and zero-copy conversions

| Need                                         | Use                                     | Notes                                                       |
|----------------------------------------------|-----------------------------------------|-------------------------------------------------------------|
| Marshal without handling the error           | `ToJSON(v)`, `ToJSONStr(v)`             | Returns empty output on error — for values that always encode |
| Unmarshal into a type                        | `FromJSON[T](b)`                        | Zero value on error                                         |
| Convert between types via their JSON form    | `CastJSON[T](v)`, `ToMap(v)`            | Partially fills on mismatch                                 |
| A type assertion with a zero-value fallback  | `Cast[T](v)`, `TryCast[T](v)`           | Equivalent                                                  |
| `[]byte`→`string` without copying            | `B2S(b)` (alias of `ByteToStr`)         | The string aliases `b`: don't modify `b` while it is in use |
| `string`→`[]byte` without copying            | `S2B(s)` (alias of `StrToByte`)         | Never write to the result; `nil` for `""`                   |
| An owned copy                                | `CloneStr(s)`, `CloneBytes(b)`          |                                                             |
| A number decodable from string, int or float | `Numeric`, `ParseNumeric(v)`            | Marshals as a string with fixed precision                   |

### Strings and numbers

| Need                         | Use                                                                                                | Notes                                                                   |
|------------------------------|----------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------|
| Parse, ignoring errors       | `StrToInt`, `StrToInt32`, `StrToInt64`, `StrToUInt`, `StrToUInt32`, `StrToUInt64`, `StrToFloat32`, `StrToFloat64` | Malformed → 0; out-of-range integers clamp to the type's bounds, floats become ±Inf; negative → 0 for unsigned. Use `strconv` when you need the error |
| Format                       | `IntToStr`, `Int32ToStr`, `Int64ToStr`, `UIntToStr`, `UInt32ToStr`, `UInt64ToStr`, `Float32ToStr`, `Float64ToStr` (`F32ToStr`, `F64ToStr`) | Floats use the shortest round-trip form, never an exponent |
| Truncate to n runes          | `StrTruncate(s, n)`                                                                                | Cuts by rune, adds no ellipsis                                          |
| Multi-line output            | `L(format, args...)` → `StrLine`, `StrLines(lines...)`                                             | One `fmt.Sprintf` per line, joined with `\n`                            |

### Case transforms

| Need                    | Use                                                         |
|-------------------------|-------------------------------------------------------------|
| `CamelCase`, `camelCase` | `ToCamel`, `ToLowerCamel`                                  |
| `snake_case`, `SCREAMING_SNAKE` | `ToSnake`, `ToScreamingSnake`, `ToSnakeWithIgnore`  |
| `kebab-case`, `SCREAMING-KEBAB` | `ToKebab`, `ToScreamingKebab`                       |
| Any delimiter           | `ToDelimited(s, '.')`, `ToScreamingDelimited(s, '.', ignore, screaming)` |

Words split on case changes and digit boundaries, so `JSONData2` → `json_data_2`.

### Slices and maps

| Need                                   | Use                                                    | Notes                                                     |
|----------------------------------------|--------------------------------------------------------|-----------------------------------------------------------|
| Transform each element                 | `Map(s, fn)`                                           | Always returns a non-nil slice (JSON `[]`, not `null`)    |
| Keep matching elements                 | `Filter(s, fn)`                                        | Non-nil, capacity clipped                                 |
| Fold to one value                      | `Reduce(fn, s)`                                        | Starts from the zero value; the function comes first      |
| Mutate in place                        | `ForEach(s, func(*T))`                                 |                                                           |
| Process in pages                       | `Paginate(s, size, func(start, end int) error)`        | Calls `fn(0, 0)` once for an empty slice                  |
| Membership                             | `Contains`, `ContainsAny`, `ContainsAll`               | `Contains` is `slices.Contains`; `ContainsAll(s, nil)` is true |
| Append only if absent                  | `AddUnique(s, v)`                                      | Linear scan                                               |
| Map keys / values → slice              | `MapKeysToArray(m)`, `MapToArray(m)`, `MapToArrayFunc(m, fn)` | Unordered; sort afterwards if order matters        |
| Slice → map                            | `ArrayToMap(s)` (by index), `ArrayToMapFunc(s, keyFn)` |                                                           |
| Slice → set                            | `ArrayToSet(s)`                                        | `map[T]struct{}`; look up with `_, ok := set[x]`          |
| Rewrite a map                          | `TransformMap(m, func(k, v) (k2, v2))`                 |                                                           |
| First present key                      | `First(m, keys...)`, `FirstOr(def, m, keys...)`        |                                                           |

### Pointers, defaults and errors

| Need                                        | Use                            | Notes                                                        |
|---------------------------------------------|--------------------------------|--------------------------------------------------------------|
| Nil-safe dereference                        | `PtrVal(p)`                    | Zero value for `nil`                                         |
| Pointer to a value                          | `new(v)` (Go 1.26+)            | `ValPtr(v)` is kept for older code; `go fix` inlines it      |
| Pointer, or `nil` for the zero value        | `ValPtrOrNil(v)`               |                                                              |
| First non-zero value                        | `Coalesce(a, b, c)`            | Evaluates every argument first — don't pass expensive calls  |
| Panic on error (startup, `MustX` wrappers)  | `Must(f())`, `Assert(err)`     |                                                              |
| Drop or replace an error                    | `Ok(f())`, `OkOr(v, err, fallback)` |                                                         |

### Struct mapping

| Need                        | Use                                                       | Notes                                  |
|-----------------------------|-----------------------------------------------------------|----------------------------------------|
| Deep struct-to-struct copy  | `DynCast[DST](src, "SrcField", "DstField", ...)`          | Panics on error                        |
| ...with custom conversions  | `DynCastOption[DST](src, converters, ...)` + `TypeConvert(fn)` → `TypeConverter` | Use instead of `jinzhu/copier` |

### Time, files and locking

| Need                                   | Use                                              | Notes                                                        |
|----------------------------------------|--------------------------------------------------|--------------------------------------------------------------|
| Current Unix seconds on a hot path     | `TimeUnix()`                                     | Up to 1s stale; starts a refresh goroutine on first call     |
| Offset a Unix timestamp                | `TimeUnixAdd(t, d)`, `TimeUnixSubtract(t, d)`    | Second resolution                                            |
| Cheap monotonic durations              | `NanoTime()`, `CPUTicks()`                       | Runtime-linked; not wall-clock time                          |
| Executable / working directory         | `GetExecDir()`, `GetExecName()`, `GetCurrentDir()` | `GetExecDir`/`GetCurrentDir` return `""` on error           |
| Copy a file                            | `CopyFile(src, dst)`, `CopyFileWithBuffer(src, dst, buf)` | OS paths only (not `fs.FS`)                         |
| YAML file I/O                          | `ReadYamlFile(path, &v)`, `WriteYamlFile(path, v)` |                                                            |
| A lock for very short critical sections | `SpinLock`                                      | Spins with `runtime.Gosched`; use `sync.Mutex` otherwise     |

### Cached struct field access — `rkit/reflector`

```go
import "github.com/clubpay/ronykit/x/rkit/reflector"
```

`reflector.Register(m, tags...)` caches a struct type's `Fields` (one `FieldInfo` per field, by name or tag) so later
lookups skip reflection. `reflector.New()` returns a `Reflector`: `Load(m, tags...)` gives the `Reflected` view
(`Obj()`, `ByTag(tag)`), and `Get`/`GetString`/`GetInt` read one field. Errors: `ErrNoField`, `ErrNotExported`,
`ErrMessageIsNotStruct`.

## Keeping this current

Every exported identifier must appear in this README and in the ronyup MCP resource
`ronyup/cmd/mcp/knowledge/resources/packages/rkit.md`; a ronyup test fails otherwise. When you add a helper, add it to
both, and consider whether `ronyup lint` (`ronyup/internal/rkitlint`) should catch hand-written copies of it.
