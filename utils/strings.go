package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	i18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

func ReplaceStrings(original string, replace map[string]string) string {
	if replace == nil {
		return original
	}
	for k, v := range replace {
		original = strings.ReplaceAll(original, k, v)
	}
	return original
}
func CurlyBraceReplaceStrings(original string, replace map[string]string) string {
	if replace == nil {
		return original
	}
	for k, v := range replace {
		original = strings.ReplaceAll(original, fmt.Sprintf("{%s}", k), v)
	}
	return original
}

func LocalizeWithInterperlate(localizer *i18n.Localizer, id string, replace map[string]string) string {
	template, err := localizer.LocalizeMessage(&i18n.Message{ID: id})
	if err != nil {
		return id
	}

	s := CurlyBraceReplaceStrings(template, replace)
	return s
}

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
