package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	i18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// ReplaceStrings performs bulk string replacement using the provided key-value map.
func ReplaceStrings(original string, replace map[string]string) string {
	if replace == nil {
		return original
	}
	for k, v := range replace {
		original = strings.ReplaceAll(original, k, v)
	}
	return original
}

// CurlyBraceReplaceStrings replaces {key} placeholders in the string with values from the map.
func CurlyBraceReplaceStrings(original string, replace map[string]string) string {
	if replace == nil {
		return original
	}
	for k, v := range replace {
		original = strings.ReplaceAll(original, fmt.Sprintf("{%s}", k), v)
	}
	return original
}

// LocalizeWithInterpolate localizes a message by ID and interpolates curly-brace placeholders.
func LocalizeWithInterpolate(localizer *i18n.Localizer, id string, replace map[string]string) string {
	template, err := localizer.LocalizeMessage(&i18n.Message{ID: id})
	if err != nil {
		return id
	}

	s := CurlyBraceReplaceStrings(template, replace)
	return s
}

// Deprecated: Use LocalizeWithInterpolate instead.
func LocalizeWithInterperlate(localizer *i18n.Localizer, id string, replace map[string]string) string {
	return LocalizeWithInterpolate(localizer, id, replace)
}

// LocalizeSimple localizes a message by ID without any placeholder interpolation.
func LocalizeSimple(localizer *i18n.Localizer, id string) string {
	return LocalizeWithInterperlate(localizer, id, nil)
}

// ReplaceEnvVars replaces environment variable placeholders in the format ${VAR_NAME}
// with their corresponding values from the environment.
// pattern should be in the format "${%s}" to match ${VAR_NAME} patterns
// Example: ReplaceEnvVars(`{"host":"${DB_HOST}"}`, "${%s}")
func ReplaceEnvVars(original string, pattern string) string {
	// Extract the format from pattern (e.g., "${%s}" -> extract prefix "${" and suffix "}")
	// Build a regex to find all occurrences
	escapedPattern := regexp.QuoteMeta(pattern)
	regexPattern := strings.ReplaceAll(escapedPattern, `%s`, `([A-Za-z0-9_-]+)`)

	re := regexp.MustCompile(regexPattern)

	result := re.ReplaceAllStringFunc(original, func(match string) string {
		// Extract the variable name from the match
		matches := re.FindStringSubmatch(match)
		if len(matches) > 1 {
			varName := matches[1]
			if value, exists := os.LookupEnv(varName); exists {
				return value
			}
		}
		// If environment variable doesn't exist, leave the placeholder unchanged
		return match
	})

	return result
}
