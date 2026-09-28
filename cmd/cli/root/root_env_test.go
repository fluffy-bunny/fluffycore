package root

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Default path (local.env) missing: not an error -- every command used to fail here.
	cmd := InitRootCmd()
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	if err := loadEnvFile(cmd); err != nil {
		t.Fatalf("missing default env file must be ignored, got %v", err)
	}

	// Explicit path missing: an error.
	cmd = InitRootCmd()
	if err := cmd.ParseFlags([]string{"--env_path=nope.env"}); err != nil {
		t.Fatal(err)
	}
	if err := loadEnvFile(cmd); err == nil {
		t.Fatal("an explicitly given env file that doesn't exist must be an error")
	}

	// Explicit path present: loaded.
	p := filepath.Join(dir, "x.env")
	if err := os.WriteFile(p, []byte("FLUFFY_ROOT_ENV_TEST=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Unsetenv("FLUFFY_ROOT_ENV_TEST") })
	cmd = InitRootCmd()
	if err := cmd.ParseFlags([]string{"--env_path=" + p}); err != nil {
		t.Fatal(err)
	}
	if err := loadEnvFile(cmd); err != nil || os.Getenv("FLUFFY_ROOT_ENV_TEST") != "1" {
		t.Fatalf("env file not loaded: err=%v", err)
	}
}
