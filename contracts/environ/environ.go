package environ

import "context"

type (
	ISingletonEnvironmentServices interface {
		// GetEnvironmentMap returns all environment variables as a map
		GetEnvironmentMap(ctx context.Context) (map[string]string, error)

		// ReplaceEnvironmentVariables replaces environment variable placeholders in the input string
		ReplaceEnvironmentVariables(ctx context.Context, input string, pattern string) (string, error)

		// GetEnvironmentVariable gets a single environment variable by key
		GetEnvironmentVariable(ctx context.Context, key string) (string, bool, error)

		// GetEnvironmentVariableOrDefault gets an environment variable or returns the default value
		GetEnvironmentVariableOrDefault(ctx context.Context, key string, defaultValue string) (string, error)

		// HasEnvironmentVariable checks if an environment variable exists
		HasEnvironmentVariable(ctx context.Context, key string) (bool, error)

		// SetEnvironmentVariable sets an environment variable (both in cache and OS)
		SetEnvironmentVariable(ctx context.Context, key string, value string) error

		// UnsetEnvironmentVariable removes an environment variable (both from cache and OS)
		UnsetEnvironmentVariable(ctx context.Context, key string) error

		// ClearCache clears the cached environment variables
		ClearCache(ctx context.Context) error

		// ReloadEnvironmentMap forces a reload of all environment variables
		ReloadEnvironmentMap(ctx context.Context) (map[string]string, error)
	}
)
