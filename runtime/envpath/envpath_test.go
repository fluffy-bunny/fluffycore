package envpath

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type innerConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
}

type nestedConfig struct {
	ServiceName string      `json:"serviceName"`
	Tracing     innerConfig `json:"tracingConfig"`
}

type sliceConfig struct {
	Tags    []string `json:"tags"`
	Numbers []int    `json:"numbers"`
}

type embeddedCore struct {
	AppName string `json:"applicationName"`
	Port    int    `json:"port"`
}

type configWithEmbed struct {
	embeddedCore
	Custom string `json:"custom"`
}

type arrayOfStructsConfig struct {
	Items []itemConfig `json:"items"`
}

type itemConfig struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func setEnvs(t *testing.T, envs map[string]string) {
	t.Helper()
	for k, v := range envs {
		t.Setenv(k, v)
	}
}

func TestApply_FlatFields(t *testing.T) {
	type flat struct {
		Host string `json:"host"`
		Port int    `json:"port"`
		TLS  bool   `json:"tls"`
	}

	cfg := flat{Host: "localhost", Port: 8080, TLS: false}
	setEnvs(t, map[string]string{
		"APP__host": "example.com",
		"APP__port": "9090",
		"APP__tls":  "true",
	})

	// flat fields have __ in them so they are picked up
	// Wait — flat fields like APP__host have one __ so they DO contain the delimiter.
	err := Apply("APP", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "example.com", cfg.Host)
	assert.Equal(t, 9090, cfg.Port)
	assert.True(t, cfg.TLS)
}

func TestApply_NestedStruct(t *testing.T) {
	cfg := nestedConfig{
		ServiceName: "default",
		Tracing:     innerConfig{Enabled: false, Endpoint: "localhost:4318"},
	}
	setEnvs(t, map[string]string{
		"TEST__tracingConfig__enabled":  "true",
		"TEST__tracingConfig__endpoint": "otel.prod:4318",
		"TEST__serviceName":             "my-service",
	})

	err := Apply("TEST", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "my-service", cfg.ServiceName)
	assert.True(t, cfg.Tracing.Enabled)
	assert.Equal(t, "otel.prod:4318", cfg.Tracing.Endpoint)
}

func TestApply_CommaSlice(t *testing.T) {
	cfg := sliceConfig{Tags: []string{"default"}}
	setEnvs(t, map[string]string{
		"SL__tags": "alpha,beta,gamma",
	})

	err := Apply("SL", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, cfg.Tags)
}

func TestApply_EmbeddedStruct(t *testing.T) {
	cfg := configWithEmbed{
		embeddedCore: embeddedCore{AppName: "old", Port: 80},
		Custom:       "old-custom",
	}
	setEnvs(t, map[string]string{
		"EMB__applicationName": "new-app",
		"EMB__port":            "443",
		"EMB__custom":          "new-custom",
	})

	err := Apply("EMB", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "new-app", cfg.AppName)
	assert.Equal(t, 443, cfg.Port)
	assert.Equal(t, "new-custom", cfg.Custom)
}

func TestApply_ArrayIndex(t *testing.T) {
	cfg := arrayOfStructsConfig{
		Items: []itemConfig{
			{Name: "first", Value: 1},
			{Name: "second", Value: 2},
		},
	}
	setEnvs(t, map[string]string{
		"ARR__items__0__name":  "updated-first",
		"ARR__items__1__value": "99",
	})

	err := Apply("ARR", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "updated-first", cfg.Items[0].Name)
	assert.Equal(t, 1, cfg.Items[0].Value)       // unchanged
	assert.Equal(t, "second", cfg.Items[1].Name) // unchanged
	assert.Equal(t, 99, cfg.Items[1].Value)
}

func TestApply_NoPrefix(t *testing.T) {
	setEnvs(t, map[string]string{
		"section__name": "new-name",
	})

	type nested struct {
		Section struct {
			Name string `json:"name"`
		} `json:"section"`
	}
	cfg := nested{}
	cfg.Section.Name = "old"

	err := Apply("", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "new-name", cfg.Section.Name)
}

func TestApply_CaseInsensitive(t *testing.T) {
	cfg := nestedConfig{
		Tracing: innerConfig{Enabled: false},
	}
	setEnvs(t, map[string]string{
		"CI__TRACINGCONFIG__ENABLED": "true",
	})

	err := Apply("CI", "__", &cfg)
	require.NoError(t, err)
	assert.True(t, cfg.Tracing.Enabled)
}

func TestApply_CaseInsensitive_MixedCasePrefix(t *testing.T) {
	type cfg struct {
		ApplicationName string `json:"applicationName"`
		Port            int    `json:"port"`
	}

	tests := []struct {
		name string
		envs map[string]string
	}{
		{"ALL_UPPER", map[string]string{"EXAMPLE__APPLICATIONNAME": "app1", "EXAMPLE__PORT": "8080"}},
		{"all_lower", map[string]string{"example__applicationname": "app1", "example__port": "8080"}},
		{"Mixed_Prefix", map[string]string{"Example__ApplicationName": "app1", "Example__Port": "8080"}},
		{"SCREAMING_with_camel_key", map[string]string{"EXAMPLE__applicationName": "app1", "EXAMPLE__port": "8080"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cfg{ApplicationName: "default", Port: 3000}
			setEnvs(t, tt.envs)

			err := Apply("EXAMPLE", "__", &c)
			require.NoError(t, err)
			assert.Equal(t, "app1", c.ApplicationName)
			assert.Equal(t, 8080, c.Port)
		})
	}
}

func TestApply_SkipsNonMatchingEnvVars(t *testing.T) {
	type simple struct {
		Name string `json:"name"`
	}
	cfg := simple{Name: "original"}
	setEnvs(t, map[string]string{
		"APP__nonexistent__field": "value",
	})

	err := Apply("APP", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "original", cfg.Name) // unchanged
}

func TestApply_NotAPointer(t *testing.T) {
	type simple struct {
		Name string `json:"name"`
	}
	cfg := simple{}
	err := Apply("APP", "__", cfg) // not a pointer
	require.Error(t, err)
}

func TestApply_FloatField(t *testing.T) {
	type withFloat struct {
		Rate float64 `json:"rate"`
	}
	cfg := withFloat{Rate: 1.0}
	setEnvs(t, map[string]string{
		"FL__rate": "3.14",
	})

	err := Apply("FL", "__", &cfg)
	require.NoError(t, err)
	assert.InDelta(t, 3.14, cfg.Rate, 0.001)
}

// Custom duration type implementing TextUnmarshaler
type testDuration time.Duration

func (d *testDuration) UnmarshalText(b []byte) error {
	parsed, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = testDuration(parsed)
	return nil
}

func TestApply_TextUnmarshaler(t *testing.T) {
	type withDuration struct {
		Timeout testDuration `json:"timeout"`
	}
	cfg := withDuration{Timeout: testDuration(time.Second)}
	setEnvs(t, map[string]string{
		"DUR__timeout": "30s",
	})

	err := Apply("DUR", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, time.Duration(cfg.Timeout))
}

func TestApply_ArrayOutOfBounds(t *testing.T) {
	cfg := arrayOfStructsConfig{
		Items: []itemConfig{{Name: "only", Value: 1}},
	}
	setEnvs(t, map[string]string{
		"OOB__items__5__name": "boom",
	})

	// Should not error — just skips
	err := Apply("OOB", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "only", cfg.Items[0].Name)
}

func TestApply_IgnoresEnvVarsWithoutDelimiter(t *testing.T) {
	type simple struct {
		Name string `json:"name"`
	}
	cfg := simple{Name: "original"}
	os.Setenv("APP_name", "should-not-apply")
	defer os.Unsetenv("APP_name")

	err := Apply("APP", "__", &cfg)
	require.NoError(t, err)
	assert.Equal(t, "original", cfg.Name) // single underscore, no __ delimiter
}
