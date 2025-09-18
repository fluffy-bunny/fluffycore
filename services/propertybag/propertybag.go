package propertybag

import (
	"sync"

	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_propertybag "github.com/fluffy-bunny/fluffycore/contracts/propertybag"
)

type (
	service struct {
		storage sync.Map
	}
)

var stemService = (*service)(nil)
var _ fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag = (*service)(nil)

func (s *service) Ctor() (fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag, error) {
	return &service{}, nil
}

// AddScopedRequestContextLoggingPropertyBag ...
func AddScopedRequestContextLoggingPropertyBag(builder di.ContainerBuilder) {
	di.AddScoped[fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag](builder, stemService.Ctor)
}

// Get gets a value from the bag
func (s *service) Get(key string) (any, bool) {
	if fluffycore_utils.IsEmptyOrNil(key) {
		return nil, false
	}
	return s.storage.Load(key)
}

// Set sets a value in the bag
func (s *service) Set(key string, value any) {
	if fluffycore_utils.IsNotEmptyOrNil(key) {
		s.storage.Store(key, value)
	}
}

// Delete deletes a value from the bag
func (s *service) Delete(key string) {
	if fluffycore_utils.IsNotEmptyOrNil(key) {
		s.storage.Delete(key)
	}
}

// Keys returns all keys in the bag
func (s *service) Keys() []string {
	keys := make([]string, 0)
	s.storage.Range(func(key any, value any) bool {
		if k, ok := key.(string); ok {
			keys = append(keys, k)
		}
		return true
	})
	return keys
}

// AsMap returns the bag as a map
func (s *service) AsMap() map[string]any {
	response := make(map[string]any)
	s.storage.Range(func(key any, value any) bool {
		if k, ok := key.(string); ok {
			response[k] = value
		}
		return true
	})
	return response
}
