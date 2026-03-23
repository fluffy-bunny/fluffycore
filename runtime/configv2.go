package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	fluffycore_contract_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	"github.com/fluffy-bunny/fluffycore/runtime/envpath"
	log "github.com/rs/zerolog/log"
)

// LoadConfigV2 loads configuration using a layered approach with zero external dependencies:
//
//	Layer 0: Go defaults already in Destination (caller sets via NewDefaultConfig())
//	Layer 1..N: JSONSources applied as sparse overlays via json.Unmarshal
//	Layer N+1: appsettings.{env}.json merged if ConfigPath is set
//	Layer N+2: Environment variables (PREFIX__section__field) applied last
func LoadConfigV2(opts *fluffycore_contract_runtime.ConfigOptionsV2) error {
	if opts.Destination == nil {
		return fmt.Errorf("configv2: Destination must not be nil")
	}

	// Layer 1..N: Apply each JSON source as a sparse overlay.
	// json.Unmarshal onto a pre-populated struct only touches fields present in JSON.
	for i, src := range opts.JSONSources {
		if len(src) == 0 {
			continue
		}
		if err := json.Unmarshal(src, opts.Destination); err != nil {
			return fmt.Errorf("configv2: json source %d: %w", i, err)
		}
	}

	// Layer N+1: Merge environment-specific JSON file if present
	environment := os.Getenv("APPLICATION_ENVIRONMENT")
	if environment != "" && opts.ConfigPath != "" {
		envFile := filepath.Join(opts.ConfigPath, "appsettings."+environment+".json")
		data, err := os.ReadFile(envFile)
		if err == nil {
			if err := json.Unmarshal(data, opts.Destination); err != nil {
				return fmt.Errorf("configv2: env config %s: %w", envFile, err)
			}
			log.Info().Str("configPath", envFile).Msg("Merged environment config")
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("configv2: reading %s: %w", envFile, err)
		}
	}

	// Layer N+2: Apply environment variable overrides
	if err := envpath.Apply(opts.EnvPrefix, "__", opts.Destination); err != nil {
		return fmt.Errorf("configv2: env overrides: %w", err)
	}

	return nil
}
