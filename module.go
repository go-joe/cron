// Package cron implements cron jobs for the Joe bot library.
// https://github.com/go-joe/joe
package cron

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/go-joe/joe"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

// A Job is a joe.Module that runs a single cron job on a given interval.
type Job struct {
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
func ScheduleEvent(schedule string, events ...interface{}) *Job {
	if len(events) == 0 {
		events = []interface{}{Event{}}
	}

	s, err := cron.Parse(schedule)
	if err != nil {
		err = fmt.Errorf("invalid cron schedule: %w", err)
	}

	return &Job{
		schedule: s,
		err:      err,
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
func ScheduleFunc(schedule string, fun func()) *Job {
	s, err := cron.Parse(schedule)
	if err != nil {
		err = fmt.Errorf("invalid cron schedule: %w", err)
	}

	return &Job{
		schedule: s,
		err:      err,
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
func ScheduleEventEvery(schedule time.Duration, events ...interface{}) *Job {
	if len(events) == 0 {
		events = []interface{}{Event{}}
	}

	return &Job{
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
func ScheduleFuncEvery(schedule time.Duration, fun func()) *Job {
	return &Job{
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
func (j *Job) Apply(conf *joe.Config) error {
	logger := conf.Logger("cron")
	events := conf.EventEmitter()
	return j.Start(logger, events)
}

// Start starts the cron job. If you are using the job as joe.Module there is no
// need to start the job explicitly. This function is useful if you want to
// manage jobs yourself if you do not pass them to the bot as joe.Module.
//
// If the job does not actually emit events, the
// passed event emitter will not be used and can be nil.
func (j *Job) Start(logger *zap.Logger, events joe.EventEmitter) error {
	if j.err != nil {
		return j.err
	}

	logger.Info("Registering new cron job",
		zap.String("typ", j.typ),
		zap.String("schedule", j.sched),
		zap.Time("next_run", j.schedule.Next(time.Now())),
	)

	job := j.fun(events)
	j.cron = cron.New()
	j.cron.ErrorLog, _ = zap.NewStdLogAt(logger, zap.ErrorLevel)
	j.cron.Schedule(j.schedule, job)
	j.cron.Start()

	return nil
}

// Close stops the cron job.
func (j *Job) Close() error {
	// The job may be nil if the used cron expression was invalid.
	if j.cron != nil {
		j.cron.Stop()
	}

	return nil
}
