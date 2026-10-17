// Package rkit holds the small helpers shared by RonyKIT modules and scaffolded services.
// Reach for it before writing a helper of your own:
//
//   - IDs and randomness: RandomID, RandomDigit, RandomInt*, SecureRandom* (random.go)
//   - JSON and casts: ToJSON, FromJSON, CastJSON, Cast; zero-copy B2S / S2B (cast.go, unsafe.go)
//   - string↔number: StrTo*, *ToStr, StrTruncate, L / StrLines (str.go)
//   - case transforms: ToCamel, ToSnake, ToKebab, ... (transform.go)
//   - slices and maps: Map, Filter, Reduce, Contains, MapKeysToArray, ArrayToSet, ... (array.go)
//   - pointers, defaults, errors: PtrVal, Coalesce, Must, Assert, Ok (generic.go)
//   - struct mapping: DynCast, DynCastOption (cast.go)
//   - time, files, locking: TimeUnix, CopyFile, ReadYamlFile, SpinLock (time.go, file.go, spinlock.go)
//
// README.md in this directory maps tasks to helpers and lists the edge cases (nil vs empty
// results, error swallowing, aliasing). In modules that require rkit, `ronyup lint` reports
// hand-written copies of the common helpers.
package rkit
