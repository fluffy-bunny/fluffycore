package singletonenvironmentservices

import (
	"context"
	"os"
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_environ "github.com/fluffy-bunny/fluffycore/contracts/environ"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
)

type (
	service struct {
		envMap sync.Map
		lock   sync.RWMutex
	}
)

var stemService = (*service)(nil)
var _ fluffycore_contracts_environ.ISingletonEnvironmentServices = (*service)(nil)

func (s *service) Ctor() (fluffycore_contracts_environ.ISingletonEnvironmentServices, error) {
	return &service{}, nil
}

// AddSingletonISingletonEnvironmentServices registers the service as a singleton
func AddSingletonISingletonEnvironmentServices(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_environ.ISingletonEnvironmentServices](builder, stemService.Ctor)
}

// GetEnvironmentMap returns all environment variables as a map
func (s *service) GetEnvironmentMap(ctx context.Context) (map[string]string, error) {
	s.lock.RLock()
	cachedMap := s.getMapFromCache()
	s.lock.RUnlock()

	if cachedMap != nil {
		return cachedMap, nil
	}

	// Cache miss - load environment variables
	s.lock.Lock()
	defer s.lock.Unlock()

	// Double-check after acquiring write lock
	cachedMap = s.getMapFromCache()
	if cachedMap != nil {
		return cachedMap, nil
	}

	// Load environment variables
	envMap := make(map[string]string)
	for _, env := range os.Environ() {
		// Split on first '=' only
		if idx := findFirstEqual(env); idx > 0 {
			key := env[:idx]
			value := env[idx+1:]
			envMap[key] = value
			s.envMap.Store(key, value)
		}
	}

	return envMap, nil
}

// ReplaceEnvironmentVariables replaces environment variable placeholders in the input string
// pattern should be in the format "${%s}" to match ${VAR_NAME} patterns
func (s *service) ReplaceEnvironmentVariables(ctx context.Context, input string, pattern string) (string, error) {
	return fluffycore_utils.ReplaceEnvVars(input, pattern), nil
}

// GetEnvironmentVariable gets a single environment variable by key
func (s *service) GetEnvironmentVariable(ctx context.Context, key string) (string, bool, error) {
	// First check cache
	if value, ok := s.envMap.Load(key); ok {
		if v, ok := value.(string); ok {
			return v, true, nil
		}
	}

	// Check OS environment
	value, exists := os.LookupEnv(key)
	if exists {
		// Cache it for future use
		s.envMap.Store(key, value)
		return value, true, nil
	}

	return "", false, nil
}

// GetEnvironmentVariableOrDefault gets an environment variable or returns the default value
func (s *service) GetEnvironmentVariableOrDefault(ctx context.Context, key string, defaultValue string) (string, error) {
	value, exists, err := s.GetEnvironmentVariable(ctx, key)
	if err != nil {
		return defaultValue, err
	}
	if !exists {
		return defaultValue, nil
	}
	return value, nil
}

// HasEnvironmentVariable checks if an environment variable exists
func (s *service) HasEnvironmentVariable(ctx context.Context, key string) (bool, error) {
	_, exists, err := s.GetEnvironmentVariable(ctx, key)
	return exists, err
}

// SetEnvironmentVariable sets an environment variable (both in cache and OS)
func (s *service) SetEnvironmentVariable(ctx context.Context, key string, value string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Set in OS
	if err := os.Setenv(key, value); err != nil {
		return err
	}

	// Update cache
	s.envMap.Store(key, value)
	return nil
}

// UnsetEnvironmentVariable removes an environment variable (both from cache and OS)
func (s *service) UnsetEnvironmentVariable(ctx context.Context, key string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Remove from OS
	if err := os.Unsetenv(key); err != nil {
		return err
	}

	// Remove from cache
	s.envMap.Delete(key)
	return nil
}

// ClearCache clears the cached environment variables
func (s *service) ClearCache(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Clear the sync.Map
	s.envMap.Range(func(key, value interface{}) bool {
		s.envMap.Delete(key)
		return true
	})

	return nil
}

// ReloadEnvironmentMap forces a reload of all environment variables
func (s *service) ReloadEnvironmentMap(ctx context.Context) (map[string]string, error) {
	// Clear the cache first
	if err := s.ClearCache(ctx); err != nil {
		return nil, err
	}

	// Reload from OS
	return s.GetEnvironmentMap(ctx)
}

// getMapFromCache returns the cached environment map if available
func (s *service) getMapFromCache() map[string]string {
	envMap := make(map[string]string)
	hasAny := false
	s.envMap.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok {
			if v, ok := value.(string); ok {
				envMap[k] = v
				hasAny = true
			}
		}
		return true
	})

	if !hasAny {
		return nil
	}
	return envMap
}

// findFirstEqual finds the index of the first '=' in the string
func findFirstEqual(s string) int {
	for i, c := range s {
		if c == '=' {
			return i
		}
	}
	return -1
}
