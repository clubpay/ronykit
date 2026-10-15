// Package ee is a typed, event-driven state runtime.
//
// States and events are plain Go structs that describe themselves: a state
// implements State (StateName), an event implements Event (EventName) and
// optionally Keyed (partition key / target id) and Labeled (labels for
// Filters). The Go type is the schema; versioning is done by declaring a new
// type with a new name, never by ee.
//
//   - StateController[S] owns one state type: CRUD, queries, and Actions.
//   - AggregateController[A] owns an aggregate: a struct whose fields are
//     states (S, *S, []S, []*S, map[string]S, map[string]*S). Developers
//     write Load; ee saves it by diffing against what Load tracked.
//   - Every event has exactly one consuming action per Env.
//   - Writes are explicit (Save/Delete) unless the Env uses WithAutoSave.
//   - The EventBus (package bus) and StorageController (package store) are
//     untyped contracts implemented by backends.
//
// State struct tags:
//
//	ID string `ee:"id"` // required, exactly one, string kind
//
// Aggregate struct tags:
//
//	Order *Order `ee:"root"` // version bumped on every save; serializes writers
//	Notes []Note `ee:"-"`    // not a member, ignored by ee
package ee
