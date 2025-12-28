package singletonenvironmentservices

import (
	"context"
	"os"
	"testing"
)

func TestGetEnvironmentMap(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR_1", "value1")
	os.Setenv("TEST_VAR_2", "value2")
	defer func() {
		os.Unsetenv("TEST_VAR_1")
		os.Unsetenv("TEST_VAR_2")
	}()

	service := &service{}
	ctx := context.Background()

	// First call - should load from environment
	envMap, err := service.GetEnvironmentMap(ctx)
	if err != nil {
		t.Fatalf("GetEnvironmentMap() error = %v", err)
	}

	if envMap["TEST_VAR_1"] != "value1" {
		t.Errorf("Expected TEST_VAR_1 = value1, got %v", envMap["TEST_VAR_1"])
	}

	if envMap["TEST_VAR_2"] != "value2" {
		t.Errorf("Expected TEST_VAR_2 = value2, got %v", envMap["TEST_VAR_2"])
	}

	// Second call - should return from cache
	envMap2, err := service.GetEnvironmentMap(ctx)
	if err != nil {
		t.Fatalf("GetEnvironmentMap() error = %v", err)
	}

	if envMap2["TEST_VAR_1"] != "value1" {
		t.Errorf("Expected cached TEST_VAR_1 = value1, got %v", envMap2["TEST_VAR_1"])
	}
}

func TestReplaceEnvironmentVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("AZUREAD_3b918868-9bff-431f-bd9c-f9896d628e6b_CLIENT_SECRET", "azure_secret_123")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("AZUREAD_3b918868-9bff-431f-bd9c-f9896d628e6b_CLIENT_SECRET")
	}()

	service := &service{}
	ctx := context.Background()

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
			name:     "Leave undefined variable unchanged",
			input:    `{"undefined":"${UNDEFINED_VAR}"}`,
			pattern:  "${%s}",
			expected: `{"undefined":"${UNDEFINED_VAR}"}`,
		},
		{
			name:     "Azure AD style variable with hyphens and GUID",
			input:    `{"clientSecret":"${AZUREAD_3b918868-9bff-431f-bd9c-f9896d628e6b_CLIENT_SECRET}"}`,
			pattern:  "${%s}",
			expected: `{"clientSecret":"azure_secret_123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ReplaceEnvironmentVariables(ctx, tt.input, tt.pattern)
			if err != nil {
				t.Fatalf("ReplaceEnvironmentVariables() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("ReplaceEnvironmentVariables() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindFirstEqual(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"KEY=VALUE", 3},
		{"KEY=VALUE=MORE", 3},
		{"NOEQUAL", -1},
		{"=VALUE", 0},
		{"", -1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := findFirstEqual(tt.input)
			if result != tt.expected {
				t.Errorf("findFirstEqual(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetEnvironmentVariable(t *testing.T) {
	os.Setenv("TEST_GET_VAR", "test_value")
	defer os.Unsetenv("TEST_GET_VAR")

	service := &service{}
	ctx := context.Background()

	// Test getting existing variable
	value, exists, err := service.GetEnvironmentVariable(ctx, "TEST_GET_VAR")
	if err != nil {
		t.Fatalf("GetEnvironmentVariable() error = %v", err)
	}
	if !exists {
		t.Error("Expected variable to exist")
	}
	if value != "test_value" {
		t.Errorf("Expected value = test_value, got %v", value)
	}

	// Test getting non-existent variable
	value, exists, err = service.GetEnvironmentVariable(ctx, "NON_EXISTENT_VAR")
	if err != nil {
		t.Fatalf("GetEnvironmentVariable() error = %v", err)
	}
	if exists {
		t.Error("Expected variable to not exist")
	}
	if value != "" {
		t.Errorf("Expected empty value, got %v", value)
	}
}

func TestGetEnvironmentVariableOrDefault(t *testing.T) {
	os.Setenv("TEST_DEFAULT_VAR", "actual_value")
	defer os.Unsetenv("TEST_DEFAULT_VAR")

	service := &service{}
	ctx := context.Background()

	// Test with existing variable
	value, err := service.GetEnvironmentVariableOrDefault(ctx, "TEST_DEFAULT_VAR", "default_value")
	if err != nil {
		t.Fatalf("GetEnvironmentVariableOrDefault() error = %v", err)
	}
	if value != "actual_value" {
		t.Errorf("Expected actual_value, got %v", value)
	}

	// Test with non-existent variable
	value, err = service.GetEnvironmentVariableOrDefault(ctx, "NON_EXISTENT_VAR", "default_value")
	if err != nil {
		t.Fatalf("GetEnvironmentVariableOrDefault() error = %v", err)
	}
	if value != "default_value" {
		t.Errorf("Expected default_value, got %v", value)
	}
}

func TestHasEnvironmentVariable(t *testing.T) {
	os.Setenv("TEST_HAS_VAR", "value")
	defer os.Unsetenv("TEST_HAS_VAR")

	service := &service{}
	ctx := context.Background()

	// Test with existing variable
	exists, err := service.HasEnvironmentVariable(ctx, "TEST_HAS_VAR")
	if err != nil {
		t.Fatalf("HasEnvironmentVariable() error = %v", err)
	}
	if !exists {
		t.Error("Expected variable to exist")
	}

	// Test with non-existent variable
	exists, err = service.HasEnvironmentVariable(ctx, "NON_EXISTENT_VAR")
	if err != nil {
		t.Fatalf("HasEnvironmentVariable() error = %v", err)
	}
	if exists {
		t.Error("Expected variable to not exist")
	}
}

func TestSetEnvironmentVariable(t *testing.T) {
	service := &service{}
	ctx := context.Background()

	// Set a new variable
	err := service.SetEnvironmentVariable(ctx, "TEST_SET_VAR", "new_value")
	if err != nil {
		t.Fatalf("SetEnvironmentVariable() error = %v", err)
	}
	defer os.Unsetenv("TEST_SET_VAR")

	// Verify it was set in OS
	osValue := os.Getenv("TEST_SET_VAR")
	if osValue != "new_value" {
		t.Errorf("Expected OS value = new_value, got %v", osValue)
	}

	// Verify it's in cache
	value, exists, err := service.GetEnvironmentVariable(ctx, "TEST_SET_VAR")
	if err != nil {
		t.Fatalf("GetEnvironmentVariable() error = %v", err)
	}
	if !exists || value != "new_value" {
		t.Errorf("Expected cached value = new_value, got %v (exists: %v)", value, exists)
	}
}

func TestUnsetEnvironmentVariable(t *testing.T) {
	service := &service{}
	ctx := context.Background()

	// Set a variable first
	os.Setenv("TEST_UNSET_VAR", "value")
	service.envMap.Store("TEST_UNSET_VAR", "value")

	// Unset it
	err := service.UnsetEnvironmentVariable(ctx, "TEST_UNSET_VAR")
	if err != nil {
		t.Fatalf("UnsetEnvironmentVariable() error = %v", err)
	}

	// Verify it's removed from OS
	osValue := os.Getenv("TEST_UNSET_VAR")
	if osValue != "" {
		t.Errorf("Expected empty OS value, got %v", osValue)
	}

	// Verify it's removed from cache
	_, exists := service.envMap.Load("TEST_UNSET_VAR")
	if exists {
		t.Error("Expected variable to be removed from cache")
	}
}

func TestClearCache(t *testing.T) {
	service := &service{}
	ctx := context.Background()

	// Add some items to cache
	service.envMap.Store("KEY1", "value1")
	service.envMap.Store("KEY2", "value2")

	// Clear cache
	err := service.ClearCache(ctx)
	if err != nil {
		t.Fatalf("ClearCache() error = %v", err)
	}

	// Verify cache is empty
	count := 0
	service.envMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	if count != 0 {
		t.Errorf("Expected empty cache, got %d items", count)
	}
}

func TestReloadEnvironmentMap(t *testing.T) {
	service := &service{}
	ctx := context.Background()

	// Set up some environment variables
	os.Setenv("TEST_RELOAD_1", "reload_value1")
	os.Setenv("TEST_RELOAD_2", "reload_value2")
	defer func() {
		os.Unsetenv("TEST_RELOAD_1")
		os.Unsetenv("TEST_RELOAD_2")
	}()

	// Load initial environment
	envMap, err := service.GetEnvironmentMap(ctx)
	if err != nil {
		t.Fatalf("GetEnvironmentMap() error = %v", err)
	}

	if envMap["TEST_RELOAD_1"] != "reload_value1" {
		t.Errorf("Expected TEST_RELOAD_1 = reload_value1, got %v", envMap["TEST_RELOAD_1"])
	}

	// Change an environment variable
	os.Setenv("TEST_RELOAD_1", "new_reload_value")

	// Without reload, should return cached value
	cachedMap, _ := service.GetEnvironmentMap(ctx)
	if cachedMap["TEST_RELOAD_1"] != "reload_value1" {
		t.Errorf("Expected cached value = reload_value1, got %v", cachedMap["TEST_RELOAD_1"])
	}

	// Reload should pick up the new value
	reloadedMap, err := service.ReloadEnvironmentMap(ctx)
	if err != nil {
		t.Fatalf("ReloadEnvironmentMap() error = %v", err)
	}

	if reloadedMap["TEST_RELOAD_1"] != "new_reload_value" {
		t.Errorf("Expected reloaded value = new_reload_value, got %v", reloadedMap["TEST_RELOAD_1"])
	}
}
