package flow

import (
	"context"
	"reflect"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
)

type SDKConfig struct {
	Logger         log.Logger
	DefaultBackend Backend
	// DeprecatingBackend should be only set when we are moving from one Temporal cluster
	// to a new Temporal cluster. This way SDK makes sure the old workflows are running
	// until they are all finished. Also, it moves all the schedulers into the new cluster.
	DeprecatingBackend Backend
}

type SDK struct {
	l   log.Logger
	b   Backend
	old Backend

	migrateCancel context.CancelFunc
	migrateDone   <-chan struct{}
}

func NewSDK(cfg SDKConfig) *SDK {
	sdk := &SDK{
		l:   cfg.Logger,
		b:   cfg.DefaultBackend,
		old: cfg.DeprecatingBackend,
	}

	return sdk
}

func (sdk *SDK) Start() error {
	err := sdk.b.Start()
	if err != nil {
		return err
	}

	if sdk.old != nil {
		err = sdk.old.Start()
		if err != nil {
			sdk.b.Stop()

			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		sdk.migrateCancel = cancel
		sdk.migrateDone = done

		go func() {
			defer close(done)

			sdk.migrateSchedulers(ctx)
		}()
	}

	return nil
}

func (sdk *SDK) Stop() {
	if sdk.migrateCancel != nil {
		sdk.migrateCancel()

		if sdk.migrateDone != nil {
			<-sdk.migrateDone
		}
	}

	sdk.b.Stop()

	if sdk.old != nil {
		sdk.old.Stop()
	}
}

func (sdk *SDK) migrateSchedulers(ctx context.Context) {
	if sdk.old == nil || sdk.b == nil {
		return
	}

	m := NewSchedulerMigrator(sdk.old, sdk.b)

	err := m.Migrate(
		ctx,
		true,
		func(ctx context.Context, sch *client.ScheduleListEntry) MigrateCheckResult {
			if len(sch.NextActionTimes) > 0 && time.Until(sch.NextActionTimes[0]) < time.Minute {
				if sdk.l != nil {
					sdk.l.Info("skipping schedule migration; next action is imminent",
						"scheduleID", sch.ID,
					)
				}

				return MigrateCheckResult{
					Ignore: true,
				}
			}

			return MigrateCheckResult{}
		},
	)
	if err != nil && sdk.l != nil {
		sdk.l.Warn("migrating schedulers", "error", err)
	}
}

func (sdk *SDK) TaskQueue() string {
	return sdk.b.TaskQueue()
}

func (sdk *SDK) Init() {
	sdk.InitWithState(EMPTY{})
}

func (sdk *SDK) InitWithState(state any) {
	registryMu.Lock()
	defer registryMu.Unlock()

	stateT := reflect.TypeOf(state)

	for stateType, w := range registeredWorkflows {
		if stateType == stateT {
			for _, t := range w {
				t.registerWithStateAny(sdk.b, state, true)

				if sdk.old != nil {
					t.registerWithStateAny(sdk.old, state, false)
				}
			}
		}
	}

	for stateType, w := range registeredActivities {
		if stateType == stateT {
			for _, t := range w {
				t.registerWithStateAny(sdk.b, state, true)

				if sdk.old != nil {
					t.registerWithStateAny(sdk.old, state, false)
				}
			}
		}
	}

	for stateType, w := range registeredActivityFactories {
		if stateType == stateT {
			for _, fn := range w {
				ent := fn(state)
				ent.registerWithStateAny(sdk.b, state, true)

				if sdk.old != nil {
					ent.registerWithStateAny(sdk.old, state, false)
				}
			}
		}
	}
}

func (sdk *SDK) UpdateWorkflowRetentionPeriod(ctx context.Context, d time.Duration) error {
	return sdk.b.UpdateWorkflowRetentionPeriod(ctx, d)
}

var _StateCtxKey struct{}

func GetState[STATE any](ctx Context) STATE {
	return ctx.Value(_StateCtxKey).(STATE)
}

type temporalEntityT interface {
	registerWithStateAny(sdk Backend, state any, setDefaultBackend bool)
}

var (
	registryMu                  sync.Mutex
	registeredWorkflows         = make(map[reflect.Type][]temporalEntityT)
	registeredActivities        = make(map[reflect.Type][]temporalEntityT)
	registeredActivityFactories = make(map[reflect.Type][]func(s any) temporalEntityT)
)
