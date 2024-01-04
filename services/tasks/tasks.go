package tasks

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_tasks "github.com/fluffy-bunny/fluffycore/contracts/tasks"
	madflojo_tasks "github.com/madflojo/tasks"
)

type service struct {
	schedular *madflojo_tasks.Scheduler
}

func init() {
	var _ fluffycore_contracts_tasks.ISingletonScheduler = (*service)(nil)
}
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
	if s.schedular != nil {
		return nil
	}
	s.schedular = madflojo_tasks.New()
	return nil
}
func (s *service) Stop() error {
	if s.schedular == nil {
		return nil
	}
	s.schedular.Stop()
	s.schedular = nil
	return nil
}
func (s *service) Add(t *madflojo_tasks.Task) (string, error) {
	return s.schedular.Add(t)
}
func (s *service) AddWithID(id string, t *madflojo_tasks.Task) error {
	return s.schedular.AddWithID(id, t)
}
func (s *service) Del(name string) {
	s.schedular.Del(name)
}

func (s *service) Lookup(name string) (*madflojo_tasks.Task, error) {
	return s.schedular.Lookup(name)
}
func (s *service) Tasks() map[string]*madflojo_tasks.Task {
	return s.schedular.Tasks()
}
