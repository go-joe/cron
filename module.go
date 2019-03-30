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

type job struct {
	cron     *cron.Cron
	schedule cron.Schedule
	fun      func(joe.EventEmitter) cron.FuncJob
	err      error

	// some meta information about the scheduled job
	typ, sched string
}

type Event struct{}

func ScheduleEvent(schedule string, events ...interface{}) joe.Module {
	if len(events) == 0 {
		events = []interface{}{Event{}}
	}

	s, err := cron.Parse(schedule)
	return &job{
		cron:     cron.New(),
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

func ScheduleFunc(schedule string, fun func()) joe.Module {
	s, err := cron.Parse(schedule)
	return &job{
		cron:     cron.New(),
		schedule: s,
		err:      errors.Wrap(err, "invalid cron schedule"),
		typ:      runtime.FuncForPC(reflect.ValueOf(fun).Pointer()).Name(),
		sched:    schedule,
		fun: func(joe.EventEmitter) cron.FuncJob {
			return fun
		},
	}
}

func ScheduleEventEvery(schedule time.Duration, events ...interface{}) joe.Module {
	if len(events) == 0 {
		events = []interface{}{Event{}}
	}

	return &job{
		cron:     cron.New(),
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

func ScheduleFuncEvery(schedule time.Duration, fun func()) joe.Module {
	return &job{
		cron:     cron.New(),
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

func (m *job) Apply(conf *joe.Config) error {
	if m.err != nil {
		return m.err
	}

	logger := conf.Logger("cron")
	m.cron.ErrorLog, _ = zap.NewStdLogAt(logger, zap.ErrorLevel)

	brain := conf.EventEmitter()
	job := m.fun(brain)

	logger.Info("Registered new cron job",
		zap.String("typ", m.typ),
		zap.String("schedule", m.sched),
		zap.Time("next_run", m.schedule.Next(time.Now())),
	)

	m.cron.Schedule(m.schedule, job)
	m.cron.Start()

	return nil
}

func (m *job) Close() error {
	m.cron.Stop()
	return nil
}
