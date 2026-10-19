package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Each of these used to come out corrupted (or, for SRV, untouched).
func TestFixupMongoConnectionString_EdgeCases(t *testing.T) {
	for original, want := range map[string]string{
		"mongodb://user:1234@localhost":                       "mongodb://user:1234@localhost/test",
		"mongodb://user:1234@localhost:27017":                 "mongodb://user:1234@localhost:27017/test",
		"mongodb://h1:27017,h2:27017":                         "mongodb://h1:27017,h2:27017/test",
		"mongodb://h1:27017/olddb?authSource=admin":           "mongodb://h1:27017/test?authSource=admin",
		"mongodb://h1:27017/olddb":                            "mongodb://h1:27017/test",
		"mongodb+srv://u:p@cluster0.example.net/?retryWrites": "mongodb+srv://u:p@cluster0.example.net/test?retryWrites",
		"mongodb+srv://cluster0.example.net":                  "mongodb+srv://cluster0.example.net/test",
		"mongodb://localhost":                                 "mongodb://localhost/test",
		"postgres://localhost:5432/db":                        "postgres://localhost:5432/db",
	} {
		require.Equal(t, want, fixupMongoConnectionString(original, "test"), original)
	}
	require.True(t, isMongoURI("mongodb+srv://x"))
	require.False(t, isMongoURI("postgres://x"))
}

type fakeMigrator struct{ calls []string }

func (f *fakeMigrator) Up() error   { f.calls = append(f.calls, "up"); return nil }
func (f *fakeMigrator) Down() error { f.calls = append(f.calls, "down"); return nil }
func (f *fakeMigrator) Steps(n int) error {
	f.calls = append(f.calls, "steps:"+itoa(n))
	return nil
}
func (f *fakeMigrator) Force(v int) error {
	f.calls = append(f.calls, "force:"+itoa(v))
	return nil
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// `force --version 16 --step 2` used to run 2 UP migrations and never force anything.
func TestRun_Dispatch(t *testing.T) {
	for _, tc := range []struct {
		md      MigrateDirection
		steps   uint64
		version int
		want    string
	}{
		{MigrateForce, 2, 16, "force:16"},
		{MigrateForce, 0, 16, "force:16"},
		{MigrateUp, 0, 0, "up"},
		{MigrateUp, 3, 0, "steps:3"},
		{MigrateDown, 0, 0, "down"},
		{MigrateDown, 3, 0, "steps:-3"},
	} {
		f := &fakeMigrator{}
		require.NoError(t, run(f, tc.md, tc.steps, tc.version))
		require.Equal(t, []string{tc.want}, f.calls)
	}
	require.Error(t, run(&fakeMigrator{}, MigrateDirection(99), 0, 0))
}
