package tasks

import (
	madflojo_tasks "github.com/madflojo/tasks"
)

type (
	IScheduler interface {
		Start() error
		Stop() error
		Add(t *madflojo_tasks.Task) (string, error)
		AddWithID(id string, t *madflojo_tasks.Task) error
		Del(name string)
		Lookup(name string) (*madflojo_tasks.Task, error)
		Tasks() map[string]*madflojo_tasks.Task
	}
	ITransientScheduler interface {
		IScheduler
	}
	ISingletonScheduler interface {
		IScheduler
	}
)
