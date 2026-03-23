// Package envpath provides ASP.NET-style environment variable overrides for Go structs.
// Environment variables use a __ (double underscore) delimiter to address nested struct fields,
// matching the json tag names. For example, MYAPP__otelConfig__tracingConfig__enabled=true
// sets Config.OTELConfig.TracingConfig.Enabled to true.
//
// Array elements can be addressed by index: MYAPP__items__0__name=foo
// Comma-separated values are split into string slices: MYAPP__tags=a,b,c
package envpath

import (
	"encoding"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const defaultDelimiter = "__"

// Apply scans environment variables for those matching the given prefix and delimiter,
// then sets the corresponding fields in dst (which must be a pointer to a struct).
// Keys are matched against json struct tags, case-insensitively.
//
// Example: with prefix="APP" and delimiter="__":
//
//	APP__server__port=8080  →  dst.Server.Port = 8080
func Apply(prefix, delimiter string, dst interface{}) error {
	if delimiter == "" {
		delimiter = defaultDelimiter
	}
	prefix = strings.TrimRight(prefix, "_")

	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("envpath: dst must be a pointer to a struct")
	}

	envs := collectEnvVars(prefix, delimiter)
	for key, value := range envs {
		segments := strings.Split(strings.ToLower(key), strings.ToLower(delimiter))
		if err := setNestedField(rv.Elem(), segments, value); err != nil {
			// Skip env vars that don't match any struct path — not an error.
			continue
		}
	}
	return nil
}

// collectEnvVars returns a map of env var keys (with prefix stripped) to values.
// The prefix is separated from the key path by the delimiter itself.
// e.g., prefix="APP", delimiter="__": APP__section__key → section__key
// e.g., prefix="APP", delimiter="__": APP__host → host (single-segment, top-level field)
// Without a prefix, only env vars containing the delimiter are collected.
func collectEnvVars(prefix, delimiter string) map[string]string {
	result := make(map[string]string)
	fullPrefix := ""
	if prefix != "" {
		fullPrefix = prefix + delimiter
	}
	for _, env := range os.Environ() {
		idx := strings.Index(env, "=")
		if idx < 0 {
			continue
		}
		key := env[:idx]
		value := env[idx+1:]

		if fullPrefix != "" {
			if !strings.HasPrefix(strings.ToUpper(key), strings.ToUpper(fullPrefix)) {
				continue
			}
			key = key[len(fullPrefix):]
			if key == "" {
				continue
			}
			result[key] = value
		} else {
			// Without a prefix, only process keys containing the delimiter
			if strings.Contains(key, delimiter) {
				result[key] = value
			}
		}
	}
	return result
}

// setNestedField walks the struct tree using json tags to find the target field,
// then sets its value.
func setNestedField(v reflect.Value, segments []string, value string) error {
	if len(segments) == 0 {
		return fmt.Errorf("no segments")
	}

	current := v
	for i, seg := range segments {
		isLast := i == len(segments)-1

		switch current.Kind() {
		case reflect.Struct:
			field, ok := findFieldByJSONTag(current, seg)
			if !ok {
				return fmt.Errorf("field %q not found", seg)
			}
			if isLast {
				return setFieldValue(field, value)
			}
			current = field

		case reflect.Slice:
			idx, err := strconv.Atoi(seg)
			if err != nil {
				return fmt.Errorf("expected array index, got %q", seg)
			}
			if idx < 0 || idx >= current.Len() {
				return fmt.Errorf("array index %d out of range (len=%d)", idx, current.Len())
			}
			elem := current.Index(idx)
			if isLast {
				return setFieldValue(elem, value)
			}
			current = elem

		case reflect.Map:
			mapKey := reflect.ValueOf(seg)
			mapVal := current.MapIndex(mapKey)
			if !mapVal.IsValid() {
				return fmt.Errorf("map key %q not found", seg)
			}
			if isLast {
				newVal := reflect.New(mapVal.Type()).Elem()
				if err := setReflectValue(newVal, value); err != nil {
					return err
				}
				current.SetMapIndex(mapKey, newVal)
				return nil
			}
			// Maps of structs need special handling — can't address into them directly.
			// For now, skip map drill-down beyond one level.
			return fmt.Errorf("cannot drill into map value at key %q", seg)

		default:
			return fmt.Errorf("cannot navigate into %s at segment %q", current.Kind(), seg)
		}
	}
	return nil
}

// findFieldByJSONTag finds a struct field whose json tag name matches the given key (case-insensitive).
// It follows embedded structs (squash/anonymous).
func findFieldByJSONTag(v reflect.Value, key string) (reflect.Value, bool) {
	t := v.Type()
	lowerKey := strings.ToLower(key)

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		field := v.Field(i)

		// Handle embedded/anonymous structs — search their fields too
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			if found, ok := findFieldByJSONTag(field, key); ok {
				return found, true
			}
			continue
		}

		jsonTag := sf.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			// Fall back to field name
			if strings.ToLower(sf.Name) == lowerKey {
				return field, true
			}
			continue
		}

		tagName := strings.Split(jsonTag, ",")[0]
		if strings.ToLower(tagName) == lowerKey {
			return field, true
		}
	}
	return reflect.Value{}, false
}

// setFieldValue sets a reflect.Value from a string, handling common types.
func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}
	return setReflectValue(field, value)
}

// setReflectValue converts a string to the appropriate type and sets the value.
func setReflectValue(v reflect.Value, s string) error {
	// Check TextUnmarshaler interface (handles custom types like Duration)
	if v.CanAddr() {
		if tu, ok := v.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return tu.UnmarshalText([]byte(s))
		}
	}

	// Check json.Unmarshaler interface
	if v.CanAddr() {
		if ju, ok := v.Addr().Interface().(json.Unmarshaler); ok {
			// Wrap string values in quotes for JSON
			jsonVal := s
			if v.Kind() == reflect.String {
				jsonVal = `"` + s + `"`
			}
			return ju.UnmarshalJSON([]byte(jsonVal))
		}
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("cannot parse %q as bool: %w", s, err)
		}
		v.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as int: %w", s, err)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as uint: %w", s, err)
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as float: %w", s, err)
		}
		v.SetFloat(f)
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.String {
			// Comma-separated string → []string
			parts := strings.Split(s, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			v.Set(reflect.ValueOf(parts))
			return nil
		}
		// For other slice types, try JSON unmarshal
		newSlice := reflect.New(v.Type())
		if err := json.Unmarshal([]byte(s), newSlice.Interface()); err != nil {
			return fmt.Errorf("cannot parse %q as %s: %w", s, v.Type(), err)
		}
		v.Set(newSlice.Elem())
	default:
		// Try JSON unmarshal as last resort
		newVal := reflect.New(v.Type())
		if err := json.Unmarshal([]byte(s), newVal.Interface()); err != nil {
			return fmt.Errorf("unsupported type %s for value %q", v.Type(), s)
		}
		v.Set(newVal.Elem())
	}
	return nil
}
