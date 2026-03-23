package runtime

import (
	"os"
	"path/filepath"
	"testing"

	fluffycore_contract_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test config types — no pointers, all value types
type testTracingConfig struct {
	Enabled      bool   `json:"enabled"`
	EndpointType string `json:"endpointType"`
	Endpoint     string `json:"endpoint"`
}

type testOTELConfig struct {
	ServiceName string            `json:"serviceName"`
	Tracing     testTracingConfig `json:"tracingConfig"`
}

type testDDConfig struct {
	TracingEnabled bool   `json:"tracingEnabled"`
	ServiceName    string `json:"serviceName"`
	Version        string `json:"version"`
}

type testCoreConfig struct {
	ApplicationName        string `json:"applicationName"`
	ApplicationEnvironment string `json:"applicationEnvironment"`
	Port                   int    `json:"port"`
	LogLevel               string `json:"logLevel"`
}

type testFullConfig struct {
	testCoreConfig
	Custom     string         `json:"customString"`
	OTELConfig testOTELConfig `json:"otelConfig"`
	DDConfig   testDDConfig   `json:"ddConfig"`
	Tags       []string       `json:"tags"`
}

func newTestDefaultConfig() testFullConfig {
	return testFullConfig{
		testCoreConfig: testCoreConfig{
			ApplicationName:        "my-app",
			ApplicationEnvironment: "development",
			Port:                   8080,
			LogLevel:               "info",
		},
		Custom: "default-custom",
		OTELConfig: testOTELConfig{
			ServiceName: "my-app",
			Tracing: testTracingConfig{
				Enabled:      false,
				EndpointType: "stdout",
				Endpoint:     "localhost:4318",
			},
		},
		DDConfig: testDDConfig{
			TracingEnabled: false,
			ServiceName:    "my-app",
			Version:        "1.0.0",
		},
		Tags: []string{"default"},
	}
}

func TestLoadConfigV2_GoDefaultsOnly(t *testing.T) {
	cfg := newTestDefaultConfig()
	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	// Everything should remain at Go defaults
	assert.Equal(t, "my-app", cfg.ApplicationName)
	assert.Equal(t, 8080, cfg.Port)
	assert.False(t, cfg.OTELConfig.Tracing.Enabled)
	assert.Equal(t, "default-custom", cfg.Custom)
}

func TestLoadConfigV2_SparseJSONOverlay(t *testing.T) {
	cfg := newTestDefaultConfig()

	// Sparse JSON — only changes 2 fields deep in the tree
	overlay := []byte(`{
		"port": 9090,
		"otelConfig": {
			"tracingConfig": {
				"enabled": true
			}
		}
	}`)

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		JSONSources: [][]byte{overlay},
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	// Changed by overlay
	assert.Equal(t, 9090, cfg.Port)
	assert.True(t, cfg.OTELConfig.Tracing.Enabled)

	// Preserved from Go defaults
	assert.Equal(t, "my-app", cfg.ApplicationName)
	assert.Equal(t, "default-custom", cfg.Custom)
	assert.Equal(t, "stdout", cfg.OTELConfig.Tracing.EndpointType)     // NOT wiped
	assert.Equal(t, "localhost:4318", cfg.OTELConfig.Tracing.Endpoint) // NOT wiped
	assert.Equal(t, "my-app", cfg.OTELConfig.ServiceName)              // NOT wiped
	assert.Equal(t, "1.0.0", cfg.DDConfig.Version)                     // NOT wiped
}

func TestLoadConfigV2_MultipleJSONLayers(t *testing.T) {
	cfg := newTestDefaultConfig()

	layer1 := []byte(`{"port": 9090, "customString": "from-layer1"}`)
	layer2 := []byte(`{"port": 3000}`) // overrides layer1's port

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		JSONSources: [][]byte{layer1, layer2},
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	assert.Equal(t, 3000, cfg.Port)                // layer2 wins
	assert.Equal(t, "from-layer1", cfg.Custom)     // from layer1, not overridden
	assert.Equal(t, "my-app", cfg.ApplicationName) // from Go defaults
}

func TestLoadConfigV2_EnvVarOverrides(t *testing.T) {
	cfg := newTestDefaultConfig()

	t.Setenv("TEST__port", "7777")
	t.Setenv("TEST__otelConfig__tracingConfig__enabled", "true")
	t.Setenv("TEST__customString", "from-env")

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		EnvPrefix:   "TEST",
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	assert.Equal(t, 7777, cfg.Port)
	assert.True(t, cfg.OTELConfig.Tracing.Enabled)
	assert.Equal(t, "from-env", cfg.Custom)
	assert.Equal(t, "my-app", cfg.ApplicationName) // unchanged
}

func TestLoadConfigV2_EnvOverridesJSON(t *testing.T) {
	cfg := newTestDefaultConfig()

	overlay := []byte(`{"port": 9090}`)
	t.Setenv("MIX__port", "1111") // env wins over JSON

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		JSONSources: [][]byte{overlay},
		EnvPrefix:   "MIX",
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	assert.Equal(t, 1111, cfg.Port) // env wins
}

func TestLoadConfigV2_AppSettingsFile(t *testing.T) {
	// Create a temp dir with an appsettings file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "appsettings.staging.json")
	err := os.WriteFile(envFile, []byte(`{
		"port": 5555,
		"otelConfig": {
			"tracingConfig": {
				"endpoint": "otel-staging:4318"
			}
		}
	}`), 0644)
	require.NoError(t, err)

	t.Setenv("APPLICATION_ENVIRONMENT", "staging")

	cfg := newTestDefaultConfig()
	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		ConfigPath:  tmpDir,
	}
	err = LoadConfigV2(opts)
	require.NoError(t, err)

	assert.Equal(t, 5555, cfg.Port)
	assert.Equal(t, "otel-staging:4318", cfg.OTELConfig.Tracing.Endpoint)
	assert.Equal(t, "stdout", cfg.OTELConfig.Tracing.EndpointType) // preserved
}

func TestLoadConfigV2_ArrayReplacement(t *testing.T) {
	cfg := newTestDefaultConfig()

	overlay := []byte(`{"tags": ["production", "critical"]}`)

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		JSONSources: [][]byte{overlay},
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	assert.Equal(t, []string{"production", "critical"}, cfg.Tags)
}

func TestLoadConfigV2_CommaSliceFromEnv(t *testing.T) {
	cfg := newTestDefaultConfig()

	t.Setenv("CSV__tags", "a,b,c")

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		EnvPrefix:   "CSV",
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	assert.Equal(t, []string{"a", "b", "c"}, cfg.Tags)
}

func TestLoadConfigV2_EmbeddedStructFields(t *testing.T) {
	cfg := newTestDefaultConfig()

	overlay := []byte(`{"applicationName": "overridden-app"}`)

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		JSONSources: [][]byte{overlay},
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	assert.Equal(t, "overridden-app", cfg.ApplicationName)
	assert.Equal(t, 8080, cfg.Port) // preserved
}

func TestLoadConfigV2_EmbeddedFieldFromEnv(t *testing.T) {
	cfg := newTestDefaultConfig()

	t.Setenv("EEMB__applicationName", "env-app")
	t.Setenv("EEMB__logLevel", "debug")

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		EnvPrefix:   "EEMB",
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)

	assert.Equal(t, "env-app", cfg.ApplicationName)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoadConfigV2_NilDestination(t *testing.T) {
	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: nil,
	}
	err := LoadConfigV2(opts)
	require.Error(t, err)
}

func TestLoadConfigV2_EmptyJSONSource(t *testing.T) {
	cfg := newTestDefaultConfig()
	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		JSONSources: [][]byte{nil, {}, []byte(`{"port": 2222}`)},
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err)
	assert.Equal(t, 2222, cfg.Port)
}

func TestLoadConfigV2_MissingAppSettingsFileIsOK(t *testing.T) {
	t.Setenv("APPLICATION_ENVIRONMENT", "nonexistent")

	cfg := newTestDefaultConfig()
	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		ConfigPath:  t.TempDir(),
	}
	err := LoadConfigV2(opts)
	require.NoError(t, err) // missing file is not an error
	assert.Equal(t, 8080, cfg.Port)
}

func TestLoadConfigV2_FullPipeline(t *testing.T) {
	// Simulate the full ASP.NET-style pipeline:
	// Go defaults → base JSON → env-specific JSON → env vars

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "appsettings.production.json")
	err := os.WriteFile(envFile, []byte(`{
		"otelConfig": {
			"tracingConfig": {
				"enabled": true,
				"endpoint": "otel-prod:4318"
			}
		}
	}`), 0644)
	require.NoError(t, err)

	t.Setenv("APPLICATION_ENVIRONMENT", "production")
	t.Setenv("FULL__port", "443")
	t.Setenv("FULL__ddConfig__tracingEnabled", "true")

	cfg := newTestDefaultConfig()
	baseJSON := []byte(`{"port": 8443, "customString": "from-base-json"}`)

	opts := &fluffycore_contract_runtime.ConfigOptionsV2{
		Destination: &cfg,
		JSONSources: [][]byte{baseJSON},
		ConfigPath:  tmpDir,
		EnvPrefix:   "FULL",
	}
	err = LoadConfigV2(opts)
	require.NoError(t, err)

	// Env var wins over JSON for port
	assert.Equal(t, 443, cfg.Port)
	// Base JSON set this, env didn't override
	assert.Equal(t, "from-base-json", cfg.Custom)
	// Env-specific JSON file set these
	assert.True(t, cfg.OTELConfig.Tracing.Enabled)
	assert.Equal(t, "otel-prod:4318", cfg.OTELConfig.Tracing.Endpoint)
	// Env var set this
	assert.True(t, cfg.DDConfig.TracingEnabled)
	// Go defaults preserved
	assert.Equal(t, "my-app", cfg.ApplicationName)
	assert.Equal(t, "stdout", cfg.OTELConfig.Tracing.EndpointType)
	assert.Equal(t, "1.0.0", cfg.DDConfig.Version)
}
