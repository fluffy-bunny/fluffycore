package js_loader

import (
	"encoding/json"
	"fmt"
	"time"

	app "github.com/maxence-charriere/go-app/v10/pkg/app"
)

/*
LoadConfigOptions contains options for loading config from JS

look at example.js.

window.isAppConfigReady = function() {}

// Helper function for WASM to get the config
window.getAppConfig = function() {}
*/
type LoadConfigOptions struct {
	IsReadyFuncName   string // i.e. "isAppConfigReady"
	GetConfigFuncName string // i.e. "getAppConfig"
}

// LoadConfigFromJS retrieves the app config that was pre-loaded by JavaScript
// This expects window.appConfig to be populated by common.js
func LoadConfigFromJS[T any](options *LoadConfigOptions) (*T, error) {
	// Wait for config to be loaded (with timeout)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for app config to load from JavaScript")
		case <-ticker.C:
			// Check if config is ready
			isReady := app.Window().Call(options.IsReadyFuncName)
			if !isReady.Bool() {
				continue
			}

			// Get the config
			configJS := app.Window().Call(options.GetConfigFuncName)
			if configJS.IsNull() || configJS.IsUndefined() {
				return nil, fmt.Errorf("app config is null or failed to load")
			}

			// Convert JS object to JSON string
			jsonStr := app.Window().Get("JSON").Call("stringify", configJS).String()

			// Unmarshal into Go struct
			var config T
			if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal app config: %w", err)
			}

			return &config, nil
		}
	}
}

// GetConfigFromJSSync attempts to get config synchronously (use only after you know it's loaded)
func GetConfigFromJSSync[T any](options *LoadConfigOptions) (*T, error) {
	// Check if config is ready
	isReady := app.Window().Call(options.IsReadyFuncName)
	if !isReady.Bool() {
		return nil, fmt.Errorf("app config is not ready yet")
	}

	// Get the config
	configJS := app.Window().Call(options.GetConfigFuncName)
	if configJS.IsNull() || configJS.IsUndefined() {
		return nil, fmt.Errorf("app config is null or failed to load")
	}

	// Convert JS object to JSON string
	jsonStr := app.Window().Get("JSON").Call("stringify", configJS).String()

	// Unmarshal into Go struct
	var config T
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app config: %w", err)
	}

	return &config, nil
}
