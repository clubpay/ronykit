package migrate

import (
	"context"
	"fmt"

	"github.com/clubpay/ronykit/flow"
	"github.com/clubpay/ronykit/kit/utils"
	"go.temporal.io/sdk/client"
)

type Scheduler struct {
	from *flow.SDK
	to   *flow.SDK
}

func NewSchedulerMigrator(from, to *flow.SDK) *Scheduler {
	return &Scheduler{
		from: from,
		to:   to,
	}
}

func (s *Scheduler) Migrate(ctx context.Context) error {
	it, err := s.from.ListSchedules(ctx, "", 100)
	if err != nil {
		return err
	}
	for it.HasNext() {
		ent, err := it.Next()
		if err != nil {
			return err
		}

		schCli := s.to.ScheduleClient()
		schH := schCli.GetHandle(ctx, ent.ID)
		schDesc, err := schH.Describe(ctx)
		if err != nil {
			return err
		}

		_, err = schCli.Create(
			ctx,
			client.ScheduleOptions{
				ID:                    ent.ID,
				Spec:                  utils.PtrVal(schDesc.Schedule.Spec),
				Action:                schDesc.Schedule.Action,
				Overlap:               schDesc.Schedule.Policy.Overlap,
				CatchupWindow:         schDesc.Schedule.Policy.CatchupWindow,
				PauseOnFailure:        schDesc.Schedule.Policy.PauseOnFailure,
				Note:                  schDesc.Schedule.State.Note,
				Paused:                schDesc.Schedule.State.Paused,
				RemainingActions:      schDesc.Schedule.State.RemainingActions,
				TypedSearchAttributes: schDesc.TypedSearchAttributes,
			},
		)
		if err != nil {
			fmt.Println("got error on creating schedule: ", err)
		}
	}

	return nil
}
