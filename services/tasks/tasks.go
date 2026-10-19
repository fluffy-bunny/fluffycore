package tasks

import (
	"errors"
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_tasks "github.com/fluffy-bunny/fluffycore/contracts/tasks"
	madflojo_tasks "github.com/madflojo/tasks"
)

type service struct {
	mu        sync.Mutex
	schedular *madflojo_tasks.Scheduler
}

// ErrNotStarted is returned by Add, AddWithID and Lookup before Start (or after Stop). They
// used to dereference the nil scheduler and panic.
var ErrNotStarted = errors.New("tasks: scheduler not started")

var _ fluffycore_contracts_tasks.ISingletonScheduler = (*service)(nil)

func AddTasksServices(cb di.ContainerBuilder) {
	addTransientScheduler(cb)
	addSingletonScheduler(cb)
}
func addTransientScheduler(cb di.ContainerBuilder) {
	di.AddTransient[fluffycore_contracts_tasks.ITransientScheduler](cb, func() fluffycore_contracts_tasks.ITransientScheduler {
		return &service{}
	})
}
func addSingletonScheduler(cb di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_tasks.ISingletonScheduler](cb, func() fluffycore_contracts_tasks.ISingletonScheduler {
		return &service{}
	})
}
func (s *service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.schedular != nil {
		return nil
	}
	s.schedular = madflojo_tasks.New()
	return nil
}
func (s *service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.schedular == nil {
		return nil
	}
	s.schedular.Stop()
	s.schedular = nil
	return nil
}

// current returns the running scheduler, or nil before Start / after Stop.
func (s *service) current() *madflojo_tasks.Scheduler {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.schedular
}

func (s *service) Add(t *madflojo_tasks.Task) (string, error) {
	sch := s.current()
	if sch == nil {
		return "", ErrNotStarted
	}
	return sch.Add(t)
}
func (s *service) AddWithID(id string, t *madflojo_tasks.Task) error {
	sch := s.current()
	if sch == nil {
		return ErrNotStarted
	}
	return sch.AddWithID(id, t)
}

// Del is a no-op when the scheduler isn't running (there is nothing to delete).
func (s *service) Del(name string) {
	if sch := s.current(); sch != nil {
		sch.Del(name)
	}
}

func (s *service) Lookup(name string) (*madflojo_tasks.Task, error) {
	sch := s.current()
	if sch == nil {
		return nil, ErrNotStarted
	}
	return sch.Lookup(name)
}

// Tasks returns an empty map when the scheduler isn't running.
func (s *service) Tasks() map[string]*madflojo_tasks.Task {
	sch := s.current()
	if sch == nil {
		return map[string]*madflojo_tasks.Task{}
	}
	return sch.Tasks()
}
