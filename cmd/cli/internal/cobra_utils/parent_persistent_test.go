package cobra_utils

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

// root (hook fails) -> middle (no hook) -> leaf. The error from two levels up used to be
// dropped by the recursive call, so the leaf ran anyway.
func TestParentHooks_PropagateErrorsThroughHooklessParents(t *testing.T) {
	boom := errors.New("boom")
	fail := func(*cobra.Command, []string) error { return boom }
	root := &cobra.Command{Use: "root", PersistentPreRunE: fail, PreRunE: fail}
	middle := &cobra.Command{Use: "middle"}
	leaf := &cobra.Command{Use: "leaf"}
	root.AddCommand(middle)
	middle.AddCommand(leaf)

	if err := ParentPersistentPreRunE(leaf, nil); !errors.Is(err, boom) {
		t.Fatalf("ParentPersistentPreRunE: want boom, got %v", err)
	}
	if err := ParentPreRunE(leaf, nil); !errors.Is(err, boom) {
		t.Fatalf("ParentPreRunE: want boom, got %v", err)
	}
}
