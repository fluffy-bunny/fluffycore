package localstorage

import (
	"encoding/json"
	"fmt"

	"github.com/fluffy-bunny/fluffycore/go-app/internal/valuecodec"
	app "github.com/maxence-charriere/go-app/v10/pkg/app"
)

// GetLocalStorage reads a value from localStorage and decodes it into type T
// For structs: expects JSON
// For primitives: reads the value directly
func GetLocalStorage[T any](key string) (T, error) {
	var result T

	storage := app.Window().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return result, fmt.Errorf("localStorage is not available")
	}

	value := storage.Call("getItem", key)
	if value.IsNull() || value.IsUndefined() {
		return result, fmt.Errorf("key %s not found in localStorage", key)
	}

	valueStr := value.String()
	if valueStr == "" {
		return result, fmt.Errorf("key %s has empty value", key)
	}

	return decodeLocalStorageValue[T](valueStr)
}

// SetLocalStorage writes a value to localStorage
// For structs: encodes as JSON
// For primitives: stores the value directly
func SetLocalStorage[T any](key string, value T) error {
	storage := app.Window().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return fmt.Errorf("localStorage is not available")
	}

	encodedValue, err := encodeLocalStorageValue(value)
	if err != nil {
		return fmt.Errorf("failed to encode value: %w", err)
	}

	storage.Call("setItem", key, encodedValue)
	return nil
}

// GetOrSetLocalStorage gets a value from localStorage, or sets and returns the default value if not found
func GetOrSetLocalStorage[T any](key string, defaultValue T) (T, error) {
	// Try to get existing value
	value, err := GetLocalStorage[T](key)
	if err == nil {
		return value, nil
	}

	// Key doesn't exist, set the default value
	err = SetLocalStorage(key, defaultValue)
	if err != nil {
		return defaultValue, fmt.Errorf("failed to set default value: %w", err)
	}

	return defaultValue, nil
}

// RemoveLocalStorage removes a key from localStorage
func RemoveLocalStorage(key string) error {
	storage := app.Window().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return fmt.Errorf("localStorage is not available")
	}

	storage.Call("removeItem", key)
	return nil
}

// ClearLocalStorage removes all keys from localStorage
func ClearLocalStorage() error {
	storage := app.Window().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return fmt.Errorf("localStorage is not available")
	}

	storage.Call("clear")
	return nil
}

// HasLocalStorageKey checks if a key exists in localStorage
func HasLocalStorageKey(key string) bool {
	storage := app.Window().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return false
	}

	value := storage.Call("getItem", key)
	return !value.IsNull() && !value.IsUndefined()
}

// GetLocalStorageKeys returns all keys in localStorage
func GetLocalStorageKeys() []string {
	storage := app.Window().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return []string{}
	}

	length := storage.Get("length")
	if length.IsUndefined() || length.IsNull() {
		return []string{}
	}

	keys := []string{}
	for i := 0; i < length.Int(); i++ {
		keyValue := storage.Call("key", i)
		if !keyValue.IsNull() && !keyValue.IsUndefined() {
			keys = append(keys, keyValue.String())
		}
	}

	return keys
}

// encodeLocalStorageValue encodes a value for storage in localStorage: primitives as text,
// anything else as JSON. A nil pointer encodes as "".
func encodeLocalStorageValue[T any](value T) (string, error) {
	v, ok := valuecodec.Deref(value)
	if !ok {
		return "", nil
	}
	if s, handled := valuecodec.EncodePrimitive(v); handled {
		return s, nil
	}
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// decodeLocalStorageValue decodes a localStorage value into type T
func decodeLocalStorageValue[T any](value string) (T, error) {
	var result T
	if value == "" {
		return result, fmt.Errorf("value is empty")
	}
	if result, handled, err := valuecodec.DecodePrimitive[T](value); handled {
		return result, err
	}
	target, get := valuecodec.NewTarget[T]()
	if err := json.Unmarshal([]byte(value), target); err != nil {
		return result, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return get(), nil
}
