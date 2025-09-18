package propertybag

import (
	"strings"
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_propertybag "github.com/fluffy-bunny/fluffycore/contracts/propertybag"
)

type (
	service struct {
		storage map[string]interface{}
		rwLock  sync.RWMutex
	}
)

var stemService = (*service)(nil)
var _ fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag = (*service)(nil)

func (s *service) Ctor() (fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag, error) {
	return &service{
		storage: make(map[string]interface{}),
	}, nil
}

// AddScopedRequestContextLoggingPropertyBag ...
func AddScopedRequestContextLoggingPropertyBag(builder di.ContainerBuilder) {
	di.AddScoped[fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag](builder, stemService.Ctor)
}

// Get gets a value from the bag
func (s *service) Get(key string) (interface{}, bool) {
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()
	val, ok := s.storage[strings.ToLower(key)]
	return val, ok
}

// Set sets a value in the bag
func (s *service) Set(key string, value interface{}) {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	s.storage[strings.ToLower(key)] = value
}

// Delete deletes a value from the bag
func (s *service) Delete(key string) {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	delete(s.storage, strings.ToLower(key))
}

// Keys returns all keys in the bag
func (s *service) Keys() []string {
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()

	keys := make([]string, 0, len(s.storage))
	for k := range s.storage {
		keys = append(keys, k)
	}
	return keys
}

// AsMap returns the bag as a map
func (s *service) AsMap() map[string]interface{} {
	s.rwLock.RLock()
	defer s.rwLock.RUnlock()

	// Create a copy of the storage map
	copy := make(map[string]interface{})
	for k, v := range s.storage {
		copy[k] = v
	}
	return copy
}
