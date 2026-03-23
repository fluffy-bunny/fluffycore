# Configuration V2

FluffyCore Config V2 is a layered configuration system inspired by [ASP.NET Configuration](https://learn.microsoft.com/en-us/aspnet/core/fundamentals/configuration/). It replaces the v1 JSON-blob+viper+mapstructure pipeline with a simpler approach that uses **zero external dependencies** — just `encoding/json`, `os`, and `reflect`.

## Motivation

V1 pain points that v2 addresses:

| Problem | V1 | V2 |
|---------|----|----|
| Defaults are a brittle JSON blob | `ConfigDefaultJSON = []byte(...)` | Type-safe Go function: `NewDefaultConfig()` |
| Pointer sub-configs require verbose JSON to "activate" | `*OTELConfig`, `*DDConfig` — nil unless JSON has the key | Value types: `OTELConfig`, `DDConfig` — always populated |
| Nil checks scattered through startup code | `if config.OTELConfig == nil { ... }` | Not needed — zero value is a valid default |
| Empty string vs unconfigured is ambiguous | `""` could mean "not set" or "intentionally empty" | Zero value = default; use `Enabled bool` for opt-in semantics |
| Two tag systems | `json:"applicationName" mapstructure:"APPLICATION_NAME"` | `json` tags only |
| External dependencies | viper, viperEx, fatih/structs, mapstructure | None (stdlib only) |

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    LoadConfigV2()                         │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  Layer 0: Go Defaults                                    │
│  ──────────────────                                      │
│  cfg := NewDefaultConfig()  // pre-populated struct      │
│                                                          │
│  Layer 1..N: Sparse JSON Overlays                        │
│  ────────────────────────────                            │
│  json.Unmarshal(overlay, &cfg)  // only changes fields   │
│                                   // present in JSON     │
│                                                          │
│  Layer N+1: Environment-Specific JSON File               │
│  ──────────────────────────────────────                  │
│  appsettings.{APPLICATION_ENVIRONMENT}.json              │
│                                                          │
│  Layer N+2: Environment Variables  (HIGHEST PRIORITY)    │
│  ────────────────────────────────                        │
│  PREFIX__section__field=value                            │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

Each layer **only overrides the fields it specifies**. Everything else is preserved from previous layers.

## Quick Start

### 1. Define your config struct (all value types)

```go
type Config struct {
    fluffycore_contracts_config.CoreConfig  // embedded, no pointer

    CustomString string                    `json:"customString"`
    OAuth2Port   int                       `json:"oauth2Port"`
    OTELConfig   otel.OTELConfig           `json:"otelConfig"`   // value, not *otel.OTELConfig
    DDConfig     ddprofiler.Config         `json:"ddConfig"`     // value, not *ddprofiler.Config
}
```

### 2. Create a defaults function

```go
func NewDefaultConfig() Config {
    return Config{
        CoreConfig: fluffycore_contracts_config.CoreConfig{
            ApplicationName: "my-app",
            PORT:            8080,
            LogLevel:        "info",
        },
        CustomString: "default-value",
        OAuth2Port:   8081,
        OTELConfig: otel.OTELConfig{
            TracingConfig: otel.TracingConfig{
                EndpointType: otel.STDOUT,
                Endpoint:     "localhost:4318",
            },
        },
    }
}
```

### 3. Implement `IStartupConfigV2` in your startup

```go
type startup struct {
    *fluffycore_runtime_otel.FluffyCoreOTELStartup

    configOptions   *fluffycore_contracts_runtime.ConfigOptions
    configOptionsV2 *fluffycore_contracts_runtime.ConfigOptionsV2
    configV2        *Config
}

// GetConfigOptions satisfies IStartup (v1 compat — required by the interface).
func (s *startup) GetConfigOptions() *fluffycore_contracts_runtime.ConfigOptions {
    defaultCfg := NewDefaultConfig()
    s.configV2 = &defaultCfg
    s.configOptions = &fluffycore_contracts_runtime.ConfigOptions{
        Destination: s.configV2,
        EnvPrefix:   "MYAPP",
    }
    return s.configOptions
}

// GetConfigOptionsV2 opts into the v2 config pipeline.
func (s *startup) GetConfigOptionsV2() *fluffycore_contracts_runtime.ConfigOptionsV2 {
    s.configOptionsV2 = &fluffycore_contracts_runtime.ConfigOptionsV2{
        Destination: s.configV2,
        EnvPrefix:   "MYAPP",
    }
    return s.configOptionsV2
}
```

The runtime automatically detects `IStartupConfigV2` and uses `LoadConfigV2` instead of `LoadConfig`.

### 4. Use the config — no nil checks

```go
func (s *startup) ConfigureServices(ctx context.Context, builder di.ContainerBuilder) {
    // V1 required:
    //   if config.OTELConfig == nil { config.OTELConfig = &otel.OTELConfig{} }
    //   if config.DDConfig == nil { config.DDConfig = &ddprofiler.Config{} }
    
    // V2: just use it
    s.configV2.OTELConfig.ServiceName = s.configV2.ApplicationName
    s.configV2.DDConfig.ServiceName = s.configV2.ApplicationName
}
```

## `ConfigOptionsV2`

```go
type ConfigOptionsV2 struct {
    // Destination is a pointer to the config struct, pre-populated with Go defaults.
    Destination interface{}

    // JSONSources are optional sparse JSON overlays applied in order.
    JSONSources [][]byte

    // ConfigPath for appsettings.{env}.json file merging.
    ConfigPath string

    // EnvPrefix for environment variable filtering (e.g., "MYAPP").
    // Env vars use the format: PREFIX__section__field
    EnvPrefix string
}
```

## Environment Variable Overrides

Environment variables use `__` (double underscore) as a path delimiter, matching the JSON key hierarchy. Keys are **case-insensitive**.

### Flat fields

```bash
MYAPP__port=9090
MYAPP__customString=hello
```

### Nested structs

```bash
MYAPP__otelConfig__tracingConfig__enabled=true
MYAPP__otelConfig__tracingConfig__endpoint=otel-prod:4318
MYAPP__ddConfig__tracingEnabled=true
```

### Embedded struct fields

Embedded struct fields (like `CoreConfig`) are addressed by their json tag directly at the parent level:

```bash
MYAPP__applicationName=my-service
MYAPP__logLevel=debug
MYAPP__port=443
```

### String slices (comma-separated)

```bash
MYAPP__jwtValidators__issuers=https://auth1.example.com,https://auth2.example.com
```

### Array element indexing

If an array is pre-populated (from Go defaults or a JSON layer), you can surgically override individual elements by index:

```bash
MYAPP__items__0__name=updated-first-item
MYAPP__items__1__value=99
```

**Important**: The array must already exist at that size from a previous layer. Environment variables cannot grow an array — they can only update existing indices.

## Sparse JSON Overlays

JSON sources only need to contain the fields you want to change. Unmentioned fields keep their previous value.

```json
{
    "port": 9090,
    "otelConfig": {
        "tracingConfig": {
            "enabled": true
        }
    }
}
```

This overlay changes `port` and `otelConfig.tracingConfig.enabled` while preserving all other fields — including `otelConfig.tracingConfig.endpoint`, `otelConfig.metricConfig`, etc.

### Multiple JSON layers

```go
s.configOptionsV2 = &fluffycore_contracts_runtime.ConfigOptionsV2{
    Destination: s.configV2,
    JSONSources: [][]byte{baseOverrides, teamOverrides},
    EnvPrefix:   "MYAPP",
}
```

Later sources win on conflicts. The full resolution order is:

```
Go defaults → JSONSources[0] → JSONSources[1] → ... → appsettings.{env}.json → env vars
```

## appsettings.{env}.json

Set `ConfigPath` and the `APPLICATION_ENVIRONMENT` env var to enable environment-specific JSON files:

```go
s.configOptionsV2 = &fluffycore_contracts_runtime.ConfigOptionsV2{
    Destination: s.configV2,
    ConfigPath:  "./config",
    EnvPrefix:   "MYAPP",
}
```

```bash
export APPLICATION_ENVIRONMENT=production
```

The runtime will look for `./config/appsettings.production.json` and merge it if present. Missing files are silently skipped.

## Arrays

| Source | Behavior |
|--------|----------|
| JSON overlay | **Whole-array replacement** — the new array completely replaces the old one |
| Environment variable (comma) | **Whole-array replacement** — `MYAPP__tags=a,b,c` replaces the entire slice |
| Environment variable (indexed) | **In-place update** — `MYAPP__items__0__name=x` updates one element |

Arrays are not surgically merged from JSON. If you include an array key in a JSON overlay, the entire array is replaced. This matches `encoding/json`'s behavior and avoids ambiguity.

## Custom Types

### `encoding.TextUnmarshaler`

Types implementing `encoding.TextUnmarshaler` are automatically handled by the env var override system. This is useful for custom duration types:

```go
type Duration time.Duration

func (d *Duration) UnmarshalText(b []byte) error {
    parsed, err := time.ParseDuration(string(b))
    if err != nil {
        return err
    }
    *d = Duration(parsed)
    return nil
}
```

```bash
MYAPP__timeout=30s
MYAPP__heartbeatInterval=5m
```

### `json.Unmarshaler`

Types implementing `json.Unmarshaler` work for both JSON layers and env var overrides.

## Backward Compatibility

V2 is fully backward compatible with v1:

- `ConfigOptions`, `LoadConfig()`, and `IStartup.GetConfigOptions()` are unchanged
- Existing startups continue to work without modification
- V2 is opt-in: implement the additional `IStartupConfigV2` interface
- The runtime checks for `IStartupConfigV2` at startup and uses the appropriate loader

Both the gRPC runtime (`runtime.Runtime`) and the Echo runtime (`echo/runtime.Runtime`) support v2 detection.

## Migration Guide

### Step 1: Convert pointer fields to value types

```go
// Before
type Config struct {
    DDConfig   *ddprofiler.Config   `json:"ddConfig"`
    OTELConfig *otel.OTELConfig     `json:"otelConfig"`
}

// After
type Config struct {
    DDConfig   ddprofiler.Config   `json:"ddConfig"`
    OTELConfig otel.OTELConfig     `json:"otelConfig"`
}
```

### Step 2: Move defaults from JSON blob to Go function

```go
// Before
var ConfigDefaultJSON = []byte(`{
    "PORT": 50051,
    "otelConfig": { "tracingConfig": { "enabled": false, ... } }
}`)

// After
func NewDefaultConfig() Config {
    return Config{
        CoreConfig: CoreConfig{PORT: 50051},
        OTELConfig: otel.OTELConfig{
            TracingConfig: otel.TracingConfig{Enabled: false, ...},
        },
    }
}
```

### Step 3: Remove `mapstructure` tags

V2 uses `json` tags exclusively. The `mapstructure:",squash"` tag on embedded structs becomes a plain embed:

```go
// Before
type Config struct {
    CoreConfig `mapstructure:",squash"`
}

// After
type Config struct {
    CoreConfig  // just embed it
}
```

### Step 4: Update startup to implement `IStartupConfigV2`

See [Quick Start](#3-implement-istartupconfiv2-in-your-startup) above.

### Step 5: Remove nil checks in `ConfigureServices`

```go
// Before
if config.OTELConfig == nil {
    config.OTELConfig = &otel.OTELConfig{}
}
config.OTELConfig.ServiceName = config.ApplicationName

// After
config.OTELConfig.ServiceName = config.ApplicationName
```

### Step 6: Pass address-of for APIs that still take pointers

Some framework APIs take pointer arguments (e.g., `SetConfig(*OTELConfig)`, `AddSingletonIProfiler(*Config)`). Use `&` on your value-type fields:

```go
s.FluffyCoreOTELStartup.SetConfig(&s.configV2.OTELConfig)
fluffycore_services_ddprofiler.AddSingletonIProfiler(builder, &s.configV2.DDConfig)
```

## Package Reference

| Package | Purpose |
|---------|---------|
| `contracts/runtime.ConfigOptionsV2` | V2 config options struct |
| `contracts/runtime.IStartupConfigV2` | Opt-in interface for v2 config |
| `runtime.LoadConfigV2()` | Layered config loader (JSON + env vars) |
| `runtime/envpath.Apply()` | ASP.NET-style `__` env var override utility |
