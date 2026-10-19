package serve

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintEnvs_DoesNotPrintValues(t *testing.T) {
	envs = []*env{{Name: "DB_PASSWORD", Value: "hunter2"}}
	t.Cleanup(func() { envs = nil })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	printEnvs()
	os.Stdout = stdout
	_ = w.Close()
	out, _ := io.ReadAll(r)

	if strings.Contains(string(out), "hunter2") {
		t.Fatalf("secret value printed: %q", out)
	}
	if !strings.Contains(string(out), "DB_PASSWORD") {
		t.Fatalf("name should still be listed: %q", out)
	}
}
