// Package ee is a typed, event-driven entity runtime.
//
// Entities and events are plain Go structs that describe themselves: an entity
// implements Entity (EntityName), an event implements Event (EventName) and
// optionally Keyed (partition key / target id) and Labeled (labels for
// Filters). The Go type is the schema; versioning is done by declaring a new
// type with a new name, never by ee.
//
//   - EntityController[S] owns one entity type: CRUD, queries, and Actions.
//   - AggregateController[A] owns an aggregate: a struct whose fields are
//     entities (S, *S, []S, []*S, map[string]S, map[string]*S). Developers
//     write Load; ee saves it by diffing against what Load tracked.
//   - Every event has exactly one consuming action per Env.
//   - Writes are explicit (Save/Delete) unless the Env uses WithAutoSave.
//     WithRequireSave fails a unit of work that mutates a loaded entity
//     without Save.
//   - Request waits for the consuming action to Reply when the bus supports
//     request/reply.
//   - The EventBus (package bus) and StorageController (package store) are
//     untyped contracts implemented by backends.
//
// Entity struct tags:
//
//	ID string `ee:"id"` // required, exactly one, string kind
//
// Aggregate struct tags:
//
//	Order *Order `ee:"root"` // version bumped on every save; serializes writers
//	Notes []Note `ee:"-"`    // not a member, ignored by ee
package ee
