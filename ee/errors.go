package ee

import (
	"errors"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

var (
	ErrNotFound          = store.ErrNotFound
	ErrConflict          = store.ErrConflict
	ErrNotSupported      = bus.ErrNotSupported
	ErrAlreadyRegistered = errors.New("ee: already registered")
	ErrNotRegistered     = errors.New("ee: not registered")
	ErrStarted           = errors.New("ee: env already started")
	ErrInvalidSchema     = errors.New("ee: invalid schema")
	ErrInvalidQuery      = errors.New("ee: invalid query")
	ErrNotMember         = errors.New("ee: state is not a member of the aggregate")
	// ErrNotLoaded is returned by Save for a state that was not loaded or
	// created in the current unit of work, so its version is unknown.
	ErrNotLoaded = errors.New("ee: state not loaded in this unit of work")
	// ErrNoKey is returned by ctx.State/ctx.Aggregate when the event does not
	// implement Keyed, so there is no target to load.
	ErrNoKey = errors.New("ee: event has no Key")
)
