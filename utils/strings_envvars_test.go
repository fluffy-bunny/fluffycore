package utils

import (
	"os"
	"testing"
)

func TestReplaceEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("API_KEY", "secret123")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("API_KEY")
	}()

	tests := []struct {
		name     string
		input    string
		pattern  string
		expected string
	}{
		{
			name:     "Replace single variable",
			input:    `{"host":"${DB_HOST}"}`,
			pattern:  "${%s}",
			expected: `{"host":"localhost"}`,
		},
		{
			name:     "Replace multiple variables",
			input:    `{"host":"${DB_HOST}","port":"${DB_PORT}"}`,
			pattern:  "${%s}",
			expected: `{"host":"localhost","port":"5432"}`,
		},
		{
			name:     "Replace with text around",
			input:    `Server is at ${DB_HOST}:${DB_PORT}`,
			pattern:  "${%s}",
			expected: `Server is at localhost:5432`,
		},
		{
			name:     "Leave undefined variable unchanged",
			input:    `{"host":"${DB_HOST}","undefined":"${UNDEFINED_VAR}"}`,
			pattern:  "${%s}",
			expected: `{"host":"localhost","undefined":"${UNDEFINED_VAR}"}`,
		},
		{
			name:     "No variables to replace",
			input:    `{"host":"static-host"}`,
			pattern:  "${%s}",
			expected: `{"host":"static-host"}`,
		},
		{
			name:     "Complex JSON with nested values",
			input:    `{"database":{"host":"${DB_HOST}","port":${DB_PORT}},"api":{"key":"${API_KEY}"}}`,
			pattern:  "${%s}",
			expected: `{"database":{"host":"localhost","port":5432},"api":{"key":"secret123"}}`,
		},
		{
			name:     "Azure AD style variable with hyphens and GUID",
			input:    `{"secret":"${AZUREAD_3b918868-9bff-431f-bd9c-f9896d628e6b_CLIENT_SECRET}"}`,
			pattern:  "${%s}",
			expected: `{"secret":"azure_secret_value"}`,
		},
	}

	// Set up Azure AD style test variable
	os.Setenv("AZUREAD_3b918868-9bff-431f-bd9c-f9896d628e6b_CLIENT_SECRET", "azure_secret_value")
	defer os.Unsetenv("AZUREAD_3b918868-9bff-431f-bd9c-f9896d628e6b_CLIENT_SECRET")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceEnvVars(tt.input, tt.pattern)
			if result != tt.expected {
				t.Errorf("ReplaceEnvVars() = %v, want %v", result, tt.expected)
			}
		})
	}
}
