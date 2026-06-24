package utils

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDuration_UnmarshalJSON_String(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{`"5m"`, 5 * time.Minute},
		{`"24h"`, 24 * time.Hour},
		{`"60s"`, 60 * time.Second},
		{`"1h30m"`, 90 * time.Minute},
		{`"0s"`, 0},
		{`"500ms"`, 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.input), &d)
			require.NoError(t, err)
			require.Equal(t, Duration(tt.expected), d)
		})
	}
}

func TestDuration_UnmarshalJSON_Integer(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"300000000000", time.Duration(300000000000)}, // 5 minutes in nanoseconds
		{"0", 0},
		{"1000000000", time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.input), &d)
			require.NoError(t, err)
			require.Equal(t, Duration(tt.expected), d)
		})
	}
}

func TestDuration_UnmarshalJSON_Invalid(t *testing.T) {
	cases := []string{
		`"notaduration"`,
		`"5x"`,
		`null`,
		`{}`,
		`[]`,
	}

	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(c), &d)
			require.Error(t, err)
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		input    Duration
		expected string
	}{
		{Duration(5 * time.Minute), `"5m0s"`},
		{Duration(24 * time.Hour), `"24h0m0s"`},
		{Duration(time.Second), `"1s"`},
		{Duration(0), `"0s"`},
		{Duration(500 * time.Millisecond), `"500ms"`},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			b, err := json.Marshal(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(b))
		})
	}
}

func TestDuration_RoundTrip(t *testing.T) {
	originals := []time.Duration{
		5 * time.Minute,
		24 * time.Hour,
		time.Second,
		500 * time.Millisecond,
		90 * time.Minute,
	}

	for _, orig := range originals {
		t.Run(orig.String(), func(t *testing.T) {
			d := Duration(orig)
			b, err := json.Marshal(d)
			require.NoError(t, err)

			var d2 Duration
			err = json.Unmarshal(b, &d2)
			require.NoError(t, err)
			require.Equal(t, d, d2)
		})
	}
}

func TestDuration_UnmarshalText(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"5m", 5 * time.Minute, false},
		{"24h", 24 * time.Hour, false},
		{"60s", 60 * time.Second, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalText([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, Duration(tt.expected), d)
			}
		})
	}
}
