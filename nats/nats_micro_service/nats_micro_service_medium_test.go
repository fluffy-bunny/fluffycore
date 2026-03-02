package nats_micro_service

import (
	"context"
	"testing"
)

func TestNATSMicroServicesContainer_ShutdownNotRegistered(t *testing.T) {
	// Shutdown when not registered should return nil
	container := &NATSMicroServicesContainer{}
	err := container.Shutdown(context.Background())
	if err != nil {
		t.Errorf("expected nil error for unregistered container, got: %v", err)
	}
}

func TestStopNATSMicroServices_EmptySlice(t *testing.T) {
	container := &NATSMicroServicesContainer{}
	err := container.stopNATSMicroServices(context.Background(), nil)
	if err != nil {
		t.Errorf("expected nil error for empty slice, got: %v", err)
	}
}

func TestConvertToStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string][]string
		expected map[string]string
	}{
		{
			name:     "empty map",
			input:    map[string][]string{},
			expected: map[string]string{},
		},
		{
			name: "single value per key",
			input: map[string][]string{
				"key1": {"val1"},
				"key2": {"val2"},
			},
			expected: map[string]string{
				"key1": "val1",
				"key2": "val2",
			},
		},
		{
			name: "multiple values takes first",
			input: map[string][]string{
				"key1": {"first", "second"},
			},
			expected: map[string]string{
				"key1": "first",
			},
		},
		{
			name: "empty values slice skipped",
			input: map[string][]string{
				"key1": {},
				"key2": {"val2"},
			},
			expected: map[string]string{
				"key2": "val2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToStringMap(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d entries, got %d", len(tt.expected), len(result))
				return
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("key %q: expected %q, got %q", k, v, result[k])
				}
			}
		})
	}
}
