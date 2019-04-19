package cron_test

import (
	"io"
	"testing"
	"time"

	"github.com/go-joe/cron"
	"github.com/go-joe/joe"
	"github.com/go-joe/joe/joetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestScheduleEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	job := cron.ScheduleEvent("* * * * * ?")

	eventReceived := make(chan bool)
	brain.RegisterHandler(func(cron.Event) {
		eventReceived <- true
	})

	err := job.Start(logger, brain)
	require.NoError(t, err)

	select {
	case <-eventReceived:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	require.NoError(t, job.Close())
	brain.Finish()
}

func TestScheduleCustomEvent(t *testing.T) {
	type CustomEvent struct{ N int }

	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	job := cron.ScheduleEvent("* * * * * ?", CustomEvent{N: 42})

	eventReceived := make(chan interface{})
	brain.RegisterHandler(func(evt CustomEvent) {
		eventReceived <- evt
	})

	err := job.Start(logger, brain)
	require.NoError(t, err)

	select {
	case got := <-eventReceived:
		assert.Equal(t, CustomEvent{42}, got)
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	require.NoError(t, job.Close())
	brain.Finish()
}

func TestScheduleEventEvery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	job := cron.ScheduleEventEvery(time.Second) // sub second durations are not yet supported

	eventReceived := make(chan bool)
	brain.RegisterHandler(func(cron.Event) {
		eventReceived <- true
	})

	err := job.Start(logger, brain)
	require.NoError(t, err)

	select {
	case <-eventReceived:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	require.NoError(t, job.Close())
	brain.Finish()
}

func TestScheduleEventEvery_CustomEvent(t *testing.T) {
	type CustomEvent struct{ N int }

	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	evt := CustomEvent{N: 1}
	job := cron.ScheduleEventEvery(time.Second, evt) // sub second durations are not yet supported

	eventReceived := make(chan interface{})
	brain.RegisterHandler(func(evt CustomEvent) {
		eventReceived <- evt
	})

	err := job.Start(logger, brain)
	require.NoError(t, err)

	select {
	case got := <-eventReceived:
		assert.Equal(t, evt, got)
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	require.NoError(t, job.Close())

	expected := []interface{}{
		CustomEvent{N: 2},
		CustomEvent{N: 3},
		CustomEvent{N: 4},
	}

	job = cron.ScheduleEventEvery(time.Second, expected...)
	err = job.Start(logger, brain)
	require.NoError(t, err)

	ok := make(chan bool)
	go func() {
		for i := 0; i < 3; i++ {
			actual := <-eventReceived
			assert.Equal(t, expected[i], actual)
		}
		ok <- true
	}()

	select {
	case <-ok:
		// great
	case <-time.After(2 * time.Second):
		t.Error("Did not see the three event in time")
	}

	require.NoError(t, job.Close())
	brain.Finish()
}

func TestJob_DoNotStartEarly(t *testing.T) {
	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	job := cron.ScheduleEventEvery(time.Hour) // next planned execution is well in the future

	eventReceived := make(chan bool)
	brain.RegisterHandler(func(cron.Event) {
		eventReceived <- true
	})

	err := job.Start(logger, brain)
	require.NoError(t, err)

	select {
	case <-eventReceived:
		t.Error("Should not have see event")
	case <-time.After(100 * time.Millisecond):
		// ok cool, lets move on
	}

	require.NoError(t, job.Close())
	brain.Finish()
}

func TestScheduleFuncEvery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	done := make(chan bool, 1)
	job := cron.ScheduleFuncEvery(time.Second, func() {
		logger.Info("Executing function")
		done <- true
	})

	err := job.Start(logger, nil)
	require.NoError(t, err)

	select {
	case <-done:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Function was not executed in time")
	}

	require.NoError(t, job.Close())
}

func TestScheduleFunc(t *testing.T) {
	logger := zaptest.NewLogger(t)
	done := make(chan bool, 1)
	job := cron.ScheduleFunc("* * * * * ?", func() {
		logger.Info("Executing function")
		done <- true
	})

	err := job.Start(logger, nil)
	require.NoError(t, err)

	select {
	case <-done:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Function was not executed in time")
	}

	require.NoError(t, job.Close())
}

func TestJob_InvalidSchedule(t *testing.T) {
	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	job := cron.ScheduleEvent("foobar")

	err := job.Start(logger, brain)
	require.EqualError(t, err, "invalid cron schedule: Expected 5 to 6 fields, found 1: foobar")
}

func TestExampleSchedule(t *testing.T) {
	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	job := cron.ScheduleEvent("0 0 * * *")

	err := job.Start(logger, brain.Brain)
	require.NoError(t, err)

	require.NoError(t, job.Close())
	brain.Finish()
}

func TestJob_Start(t *testing.T) {
	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	job := cron.ScheduleEvent("* * * * * ?")

	eventReceived := make(chan bool)
	brain.RegisterHandler(func(cron.Event) {
		eventReceived <- true
	})

	err := job.Start(logger, brain.Brain)
	require.NoError(t, err)

	select {
	case <-eventReceived:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	require.NoError(t, job.Close())
	brain.Finish()
}

func TestJob_Module(t *testing.T) {
	logger := zaptest.NewLogger(t)
	brain := joetest.NewBrain(t)
	var job joe.Module = cron.ScheduleEvent("* * * * * ?")

	eventReceived := make(chan bool)
	brain.RegisterHandler(func(cron.Event) {
		eventReceived <- true
	})

	conf := joe.NewConfig(logger, brain.Brain, nil)
	err := job.Apply(&conf)
	require.NoError(t, err)

	select {
	case <-eventReceived:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	require.NoError(t, job.(io.Closer).Close())
	brain.Finish()
}
