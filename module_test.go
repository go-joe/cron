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

func TestModule_ScheduleEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	b := joetest.NewBrain(t)
	m := cron.ScheduleEvent("* * * * * ?")

	eventReceived := make(chan bool)
	b.RegisterHandler(func(cron.Event) {
		eventReceived <- true
	})

	conf := joe.NewConfig(logger, b.Brain, nil)
	require.NoError(t, m.Apply(&conf))

	select {
	case <-eventReceived:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	_ = m.(io.Closer).Close()
	b.Finish()
}

func TestModule_ScheduleCustomEvent(t *testing.T) {
	type CustomEvent struct{ N int }

	logger := zaptest.NewLogger(t)
	b := joetest.NewBrain(t)
	m := cron.ScheduleEvent("* * * * * ?", CustomEvent{N: 42})

	eventReceived := make(chan interface{})
	b.RegisterHandler(func(evt CustomEvent) {
		eventReceived <- evt
	})

	conf := joe.NewConfig(logger, b.Brain, nil)
	require.NoError(t, m.Apply(&conf))

	select {
	case got := <-eventReceived:
		assert.Equal(t, CustomEvent{42}, got)
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	_ = m.(io.Closer).Close()
	b.Finish()
}

func TestModule_ScheduleEventEvery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	b := joetest.NewBrain(t)
	m := cron.ScheduleEventEvery(time.Second) // sub second durations are not yet supported

	eventReceived := make(chan bool)
	b.RegisterHandler(func(cron.Event) {
		eventReceived <- true
	})

	conf := joe.NewConfig(logger, b.Brain, nil)
	require.NoError(t, m.Apply(&conf))

	select {
	case <-eventReceived:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	_ = m.(io.Closer).Close()
	b.Finish()
}

func TestModule_ScheduleEventEvery_CustomEvent(t *testing.T) {
	type CustomEvent struct{ N int }

	logger := zaptest.NewLogger(t)
	b := joetest.NewBrain(t)
	evt := CustomEvent{N: 1}
	m := cron.ScheduleEventEvery(time.Second, evt) // sub second durations are not yet supported

	eventReceived := make(chan interface{})
	// TODO: why do we force the event to be a struct?
	// TODO: any registration errors are hidden by the test bot because they are only returned if the handler is registered before the bot is started
	b.RegisterHandler(func(evt CustomEvent) {
		eventReceived <- evt
	})

	conf := joe.NewConfig(logger, b.Brain, nil)
	require.NoError(t, m.Apply(&conf))

	select {
	case got := <-eventReceived:
		assert.Equal(t, evt, got)
	case <-time.After(2 * time.Second):
		t.Error("Did not see event")
	}

	_ = m.(io.Closer).Close()

	expected := []interface{}{
		CustomEvent{N: 2},
		CustomEvent{N: 3},
		CustomEvent{N: 4},
	}
	m = cron.ScheduleEventEvery(time.Second, expected...)
	require.NoError(t, m.Apply(&conf))

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

	_ = m.(io.Closer).Close()
	b.Finish()
}

func TestModule_DoNotStartEarly(t *testing.T) {
	logger := zaptest.NewLogger(t)
	b := joetest.NewBrain(t)
	m := cron.ScheduleEventEvery(time.Hour) // next planned execution is well in the future

	eventReceived := make(chan bool)
	b.RegisterHandler(func(cron.Event) {
		eventReceived <- true
	})

	conf := joe.NewConfig(logger, b.Brain, nil)
	require.NoError(t, m.Apply(&conf))

	select {
	case <-eventReceived:
		t.Error("Should not have see event")
	case <-time.After(100 * time.Millisecond):
		// ok cool, lets move on
	}

	_ = m.(io.Closer).Close()
	b.Finish()
}

func TestModule_ScheduleFuncEvery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	done := make(chan bool, 1)
	m := cron.ScheduleFuncEvery(time.Second, func() {
		logger.Info("Executing function")
		done <- true
	})

	conf := joe.NewConfig(logger, nil, nil)
	require.NoError(t, m.Apply(&conf))

	select {
	case <-done:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Function was not executed in time")
	}

	_ = m.(io.Closer).Close()
}

func TestModule_ScheduleFunc(t *testing.T) {
	logger := zaptest.NewLogger(t)
	done := make(chan bool, 1)
	m := cron.ScheduleFunc("* * * * * ?", func() {
		logger.Info("Executing function")
		done <- true
	})

	conf := joe.NewConfig(logger, nil, nil)
	require.NoError(t, m.Apply(&conf))

	select {
	case <-done:
		// ok cool, we can stop the cron job now
	case <-time.After(2 * time.Second):
		t.Error("Function was not executed in time")
	}

	_ = m.(io.Closer).Close()
}

func TestModule_InvalidSchedule(t *testing.T) {
	logger := zaptest.NewLogger(t)
	b := joetest.NewBrain(t)
	m := cron.ScheduleEvent("foobar")

	conf := joe.NewConfig(logger, b.Brain, nil)
	require.EqualError(t, m.Apply(&conf), "invalid cron schedule: Expected 5 to 6 fields, found 1: foobar")
}
