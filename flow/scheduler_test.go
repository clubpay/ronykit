package flow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.temporal.io/sdk/client"
)

func TestScheduleCalendarKeepsZeroClockFields(t *testing.T) {
	spec := ScheduleSpec{
		Calendars: []ScheduleCalendarSpec{
			{Hour: 9, Minute: 0, Second: 0},
		},
	}.toScheduleSpec()

	if len(spec.Calendars) != 1 {
		t.Fatalf("calendars %d", len(spec.Calendars))
	}

	cal := spec.Calendars[0]
	if len(cal.Hour) != 1 || cal.Hour[0].Start != 9 {
		t.Fatalf("hour %#v", cal.Hour)
	}
	if len(cal.Minute) != 1 || cal.Minute[0].Start != 0 {
		t.Fatalf("minute %#v", cal.Minute)
	}
	if len(cal.Second) != 1 || cal.Second[0].Start != 0 {
		t.Fatalf("second %#v", cal.Second)
	}
}

func TestScheduleCalendarDaysOfWeekIncludesSunday(t *testing.T) {
	spec := ScheduleSpec{
		Calendars: []ScheduleCalendarSpec{
			{Hour: 9, DaysOfWeek: []time.Weekday{time.Sunday, time.Monday}},
		},
	}.toScheduleSpec()

	days := spec.Calendars[0].DayOfWeek
	if len(days) != 2 || days[0].Start != int(time.Sunday) || days[1].Start != int(time.Monday) {
		t.Fatalf("days %#v", days)
	}
}

func TestScheduleSpecTimezoneFallback(t *testing.T) {
	req := CreateScheduleRequest{
		TimezoneName: "Europe/Istanbul",
		Spec:         ScheduleSpec{},
	}
	spec := req.Spec.toScheduleSpec()
	if spec.TimeZoneName != "" {
		t.Fatalf("expected empty timezone from spec, got %q", spec.TimeZoneName)
	}
}

type schedStore struct {
	items map[string]*client.ScheduleDescription
}

type fakeSchedClient struct {
	store *schedStore
}

type fakeSchedHandle struct {
	id    string
	store *schedStore
}

type fakeSchedIter struct {
	items []*client.ScheduleListEntry
	i     int
}

func (c *fakeSchedClient) Create(_ context.Context, options client.ScheduleOptions) (client.ScheduleHandle, error) {
	c.store.items[options.ID] = &client.ScheduleDescription{
		Schedule: client.Schedule{
			Action: options.Action,
			Spec:   &options.Spec,
		},
	}

	return &fakeSchedHandle{id: options.ID, store: c.store}, nil
}

func (c *fakeSchedClient) List(context.Context, client.ScheduleListOptions) (client.ScheduleListIterator, error) {
	items := make([]*client.ScheduleListEntry, 0, len(c.store.items))
	for id := range c.store.items {
		items = append(items, &client.ScheduleListEntry{ID: id})
	}

	return &fakeSchedIter{items: items}, nil
}

func (c *fakeSchedClient) GetHandle(_ context.Context, scheduleID string) client.ScheduleHandle {
	return &fakeSchedHandle{id: scheduleID, store: c.store}
}

func (h *fakeSchedHandle) GetID() string { return h.id }

func (h *fakeSchedHandle) Delete(context.Context) error {
	delete(h.store.items, h.id)

	return nil
}

func (h *fakeSchedHandle) Backfill(context.Context, client.ScheduleBackfillOptions) error {
	return nil
}

func (h *fakeSchedHandle) Update(context.Context, client.ScheduleUpdateOptions) error {
	return nil
}

func (h *fakeSchedHandle) Describe(context.Context) (*client.ScheduleDescription, error) {
	desc, ok := h.store.items[h.id]
	if !ok {
		return nil, errors.New("not found")
	}

	return desc, nil
}

func (h *fakeSchedHandle) Trigger(context.Context, client.ScheduleTriggerOptions) error {
	return nil
}

func (h *fakeSchedHandle) Pause(context.Context, client.SchedulePauseOptions) error {
	return nil
}

func (h *fakeSchedHandle) Unpause(context.Context, client.ScheduleUnpauseOptions) error {
	return nil
}

func (it *fakeSchedIter) HasNext() bool {
	return it.i < len(it.items)
}

func (it *fakeSchedIter) Next() (*client.ScheduleListEntry, error) {
	ent := it.items[it.i]
	it.i++

	return ent, nil
}

func TestSchedulerMigratorGivesUpOnAlwaysIgnored(t *testing.T) {
	fromStore := &schedStore{items: map[string]*client.ScheduleDescription{
		"keep": {Schedule: client.Schedule{Spec: &client.ScheduleSpec{}}},
	}}
	toStore := &schedStore{items: map[string]*client.ScheduleDescription{}}

	rounds := 0
	m := NewSchedulerMigrator(
		&mockBackend{sched: &fakeSchedClient{store: fromStore}},
		&mockBackend{sched: &fakeSchedClient{store: toStore}},
	).WithRetry(time.Millisecond, 3)

	err := m.Migrate(context.Background(), true, func(context.Context, *client.ScheduleListEntry) MigrateCheckResult {
		rounds++

		return MigrateCheckResult{Ignore: true}
	})
	if err == nil {
		t.Fatal("expected an error naming the schedules left behind")
	}
	if !strings.Contains(err.Error(), "keep") {
		t.Fatalf("error should name the pending schedule, got %v", err)
	}
	if rounds != 3 {
		t.Fatalf("expected 3 rounds, got %d", rounds)
	}
	if _, ok := fromStore.items["keep"]; !ok {
		t.Fatal("ignored schedule was deleted from source")
	}
	if _, ok := toStore.items["keep"]; ok {
		t.Fatal("ignored schedule was copied to dest")
	}
}

func TestSchedulerMigratorRetriesIgnoredSchedule(t *testing.T) {
	fromStore := &schedStore{items: map[string]*client.ScheduleDescription{
		"later": {Schedule: client.Schedule{
			Spec:   &client.ScheduleSpec{},
			Action: &client.ScheduleWorkflowAction{Workflow: "Wf"},
		}},
	}}
	toStore := &schedStore{items: map[string]*client.ScheduleDescription{}}

	m := NewSchedulerMigrator(
		&mockBackend{sched: &fakeSchedClient{store: fromStore}},
		&mockBackend{sched: &fakeSchedClient{store: toStore}},
	).WithRetry(time.Millisecond, 5)

	calls := 0
	err := m.Migrate(context.Background(), true, func(context.Context, *client.ScheduleListEntry) MigrateCheckResult {
		calls++

		return MigrateCheckResult{Ignore: calls == 1}
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := toStore.items["later"]; !ok {
		t.Fatal("schedule skipped on the first round was never migrated")
	}
	if _, ok := fromStore.items["later"]; ok {
		t.Fatal("source schedule still present")
	}
}

func TestSchedulerMigratorCopiesAndDeletes(t *testing.T) {
	fromStore := &schedStore{items: map[string]*client.ScheduleDescription{
		"move": {Schedule: client.Schedule{
			Spec:   &client.ScheduleSpec{TimeZoneName: "UTC"},
			Action: &client.ScheduleWorkflowAction{Workflow: "Wf"},
		}},
	}}
	toStore := &schedStore{items: map[string]*client.ScheduleDescription{}}

	m := NewSchedulerMigrator(
		&mockBackend{sched: &fakeSchedClient{store: fromStore}},
		&mockBackend{sched: &fakeSchedClient{store: toStore}},
	)

	if err := m.Migrate(context.Background(), true, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := fromStore.items["move"]; ok {
		t.Fatal("source schedule still present")
	}
	if _, ok := toStore.items["move"]; !ok {
		t.Fatal("dest schedule missing")
	}
}

func TestSchedulerMigratorRespectsCancel(t *testing.T) {
	fromStore := &schedStore{items: map[string]*client.ScheduleDescription{
		"move": {Schedule: client.Schedule{Spec: &client.ScheduleSpec{}}},
	}}
	toStore := &schedStore{items: map[string]*client.ScheduleDescription{}}

	m := NewSchedulerMigrator(
		&mockBackend{sched: &fakeSchedClient{store: fromStore}},
		&mockBackend{sched: &fakeSchedClient{store: toStore}},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := m.Migrate(ctx, true, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled, got %v", err)
	}
}
