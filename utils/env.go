// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// copied and modified from datadog/dd-trace-go/internal/env.go
package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	// add zerolog
	"github.com/rs/zerolog/log"
)

// StringEnv returns the value of an environment variable, or def otherwise.
func StringEnv(key, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	if IsEmptyOrNil(v) {
		return def
	}
	return v
}

// BoolEnv returns the parsed boolean value of an environment variable, or
// def otherwise.
func BoolEnv(key string, def bool) bool {
	vv, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	v, err := strconv.ParseBool(vv)
	if err != nil {
		log.Warn().Msgf("Non-boolean value for env var %s, defaulting to %t. Parse failed with error: %v", key, def, err)
		return def
	}
	return v
}

// IntEnv returns the parsed int value of an environment variable, or
// def otherwise.
func IntEnv(key string, def int) int {
	vv, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	v, err := strconv.Atoi(vv)
	if err != nil {
		log.Warn().Msgf("Non-integer value for env var %s, defaulting to %d. Parse failed with error: %v", key, def, err)
		return def
	}
	return v
}

// DurationEnv returns the parsed duration value of an environment variable, or
// def otherwise.
func DurationEnv(key string, def time.Duration) time.Duration {
	vv, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	v, err := time.ParseDuration(vv)
	if err != nil {
		log.Warn().Msgf("Non-duration value for env var %s, defaulting to %d. Parse failed with error: %v", key, def, err)
		return def
	}
	return v
}

// Env Replace in a string buffer.
// ReplaceEnv("Hello ${USER}", "${%s}")
// enumerates all env variables, turns them into the ${} format and replaces them in the string.
func ReplaceEnv(original string, format string) string {
	for _, env := range os.Environ() {
		key, value := strings.Split(env, "=")[0], strings.Split(env, "=")[1]
		original = strings.Replace(original, fmt.Sprintf(format, key), value, -1)
	}
	return original
}
