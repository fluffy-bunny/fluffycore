package tasks

import (
	"errors"
	"testing"
	"time"

	madflojo_tasks "github.com/madflojo/tasks"
	"github.com/stretchr/testify/require"
)

func task() *madflojo_tasks.Task {
	return &madflojo_tasks.Task{Interval: time.Hour, TaskFunc: func() error { return nil }}
}

// Every method used to panic on the nil scheduler before Start / after Stop.
func TestScheduler_NotStartedDoesNotPanic(t *testing.T) {
	s := &service{}
	_, err := s.Add(task())
	require.True(t, errors.Is(err, ErrNotStarted))
	require.ErrorIs(t, s.AddWithID("x", task()), ErrNotStarted)
	_, err = s.Lookup("x")
	require.ErrorIs(t, err, ErrNotStarted)
	s.Del("x")
	require.Empty(t, s.Tasks())

	require.NoError(t, s.Start())
	id, err := s.Add(task())
	require.NoError(t, err)
	require.Len(t, s.Tasks(), 1)
	s.Del(id)
	require.NoError(t, s.Stop())
	_, err = s.Add(task())
	require.ErrorIs(t, err, ErrNotStarted, "after Stop too")
}
