package flow

import (
	"context"
	"reflect"
	"time"

	"github.com/clubpay/ronykit/boxship/pkg/log"
	"go.temporal.io/sdk/client"
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
			return err
		}

		go sdk.migrateSchedulers()
	}

	return nil
}

func (sdk *SDK) Stop() {
	sdk.b.Stop()
	if sdk.old != nil {
		sdk.old.Stop()
	}
}

func (sdk *SDK) migrateSchedulers() {
	if sdk.old == nil {
		return
	}

	m := NewSchedulerMigrator(sdk.old, sdk.b)
	err := m.Migrate(
		context.Background(),
		true,
		func(ctx context.Context, sch *client.ScheduleListEntry) MigrateCheckResult {
			if len(sch.NextActionTimes) > 0 && sch.NextActionTimes[0].Sub(time.Now()) < time.Minute {
				return MigrateCheckResult{
					Ignore: true,
				}
			}

			return MigrateCheckResult{}
		},
	)
	if err != nil {
		sdk.l.Warnf("got error on migrating schedulers: %v", err)
	}
}

func (sdk *SDK) TaskQueue() string {
	return sdk.b.TaskQueue()
}

func (sdk *SDK) Init() {
	sdk.InitWithState(EMPTY{})
}

func (sdk *SDK) InitWithState(state any) {
	for stateType, w := range registeredWorkflows {
		if stateType == reflect.TypeOf(state) {
			for _, t := range w {
				t.registerWithStateAny(sdk.b, state, true)
				if sdk.old != nil {
					t.registerWithStateAny(sdk.old, state, false)
				}
			}
		}
	}
	for stateType, w := range registeredActivities {
		if stateType == reflect.TypeOf(state) {
			for _, t := range w {
				t.registerWithStateAny(sdk.b, state, true)
				if sdk.old != nil {
					t.registerWithStateAny(sdk.old, state, false)
				}
			}
		}
	}
}

var _StateCtxKey = struct{}{}

func GetState[STATE any](ctx Context) STATE {
	return ctx.Value(_StateCtxKey).(STATE)
}

type temporalEntityT interface {
	registerWithStateAny(sdk Backend, state any, setDefaultBackend bool)
}

var (
	registeredWorkflows  = make(map[reflect.Type][]temporalEntityT)
	registeredActivities = make(map[reflect.Type][]temporalEntityT)
)
