package cron

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/go-joe/joe"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

// job is a joe.Module that runs a single cron job on a given interval.
type job struct {
	cron     *cron.Cron
	schedule cron.Schedule
	fun      func(joe.EventEmitter) cron.FuncJob
	err      error // to defer error handling until joe.Module is loaded

	// some meta information about the scheduled job
	typ, sched string
}

// Event is the event that the ScheduleEvent(…) type functions emit if no custom
// event was passed as argument. It can be useful to implement simple jobs that
// do not require any context but just a schedule that triggers them at an interval.
type Event struct{}

// ScheduleEvent creates a joe.Module that emits one or many events on a given
// cron schedule (e.g. "0 0 * * *"). If the passed schedule is not a valid cron
// schedule as accepted by https://godoc.org/github.com/robfig/cron the error
// will be returned when the bot is started.
//
// You can execute this function with only a schedule but no events. In this
// case the job will emit an instance of the cron.Event type that is defined in
// this package. Otherwise all passed events are emitted on the schedule.
func ScheduleEvent(schedule string, events ...interface{}) joe.Module {
	if len(events) == 0 {
		events = []interface{}{Event{}}
	}

	s, err := cron.Parse(schedule)
	return &job{
		schedule: s,
		err:      errors.Wrap(err, "invalid cron schedule"),
		typ:      eventsString(events),
		sched:    schedule,
		fun: func(brain joe.EventEmitter) cron.FuncJob {
			return func() {
				for _, event := range events {
					brain.Emit(event)
				}
			}
		},
	}
}

// ScheduleFunc creates a joe.Module that runs the given function on a given
// cron schedule (e.g. "0 0 * * *"). If the passed schedule is not a valid cron
// schedule as accepted by https://godoc.org/github.com/robfig/cron the error
// will be returned when the bot is started.
func ScheduleFunc(schedule string, fun func()) joe.Module {
	s, err := cron.Parse(schedule)
	return &job{
		schedule: s,
		err:      errors.Wrap(err, "invalid cron schedule"),
		typ:      runtime.FuncForPC(reflect.ValueOf(fun).Pointer()).Name(),
		sched:    schedule,
		fun: func(joe.EventEmitter) cron.FuncJob {
			return fun
		},
	}
}

// ScheduleEventEvery creates a joe.Module that emits one or many events on a
// given interval (e.g. every hour). The minimum duration is one second and any
// smaller durations will be rounded up to that.
//
// You can execute this function with only a schedule but no events. In this
// case the job will emit an instance of the cron.Event type that is defined in
// this package. Otherwise all passed events are emitted on the schedule.
func ScheduleEventEvery(schedule time.Duration, events ...interface{}) joe.Module {
	if len(events) == 0 {
		events = []interface{}{Event{}}
	}

	return &job{
		schedule: cron.Every(schedule),
		typ:      eventsString(events),
		sched:    fmt.Sprintf("@every %s", schedule),
		fun: func(brain joe.EventEmitter) cron.FuncJob {
			return func() {
				for _, event := range events {
					brain.Emit(event)
				}
			}
		},
	}
}

// ScheduleFuncEvery creates a joe.Module that runs the given function on a
// given interval (e.g. every hour). The minimum duration is one second and any
// smaller durations will be rounded up to that.
func ScheduleFuncEvery(schedule time.Duration, fun func()) joe.Module {
	return &job{
		schedule: cron.Every(schedule),
		typ:      runtime.FuncForPC(reflect.ValueOf(fun).Pointer()).Name(),
		sched:    fmt.Sprintf("@every %s", schedule),
		fun: func(joe.EventEmitter) cron.FuncJob {
			return fun
		},
	}
}

func eventsString(events []interface{}) string {
	var typ string
	for i, evt := range events {
		if i > 0 {
			typ += ", "
		}
		typ += fmt.Sprintf("%T", evt)
	}

	return typ
}

// Apply implements joe.Module by starting a new cron job that may use the event
// emitter from the configuration (if it actually emits events). Jobs that only
// run functions will only require a logger.
func (m *job) Apply(conf *joe.Config) error {
	if m.err != nil {
		return m.err
	}

	logger := conf.Logger("cron")

	brain := conf.EventEmitter()
	job := m.fun(brain)

	logger.Info("Registered new cron job",
		zap.String("typ", m.typ),
		zap.String("schedule", m.sched),
		zap.Time("next_run", m.schedule.Next(time.Now())),
	)

	m.cron = cron.New()
	m.cron.ErrorLog, _ = zap.NewStdLogAt(logger, zap.ErrorLevel)
	m.cron.Schedule(m.schedule, job)
	m.cron.Start()

	return nil
}

// Close stops the cron job.
func (m *job) Close() error {
	m.cron.Stop()
	return nil
}
