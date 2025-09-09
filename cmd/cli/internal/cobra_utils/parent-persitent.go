package cobra_utils

import "github.com/spf13/cobra"

func ParentPersistentPreRunE(cmd *cobra.Command, args []string) error {
	parent := cmd.Parent()
	if parent != nil {
		if parent.PersistentPreRunE != nil {
			err := parent.PersistentPreRunE(parent, args)
			if err != nil {
				return err
			}
		} else {
			ParentPersistentPreRunE(parent, args)
		}
	}

	return nil
}

func ParentPreRunE(cmd *cobra.Command, args []string) error {
	parent := cmd.Parent()
	if parent != nil {
		if parent.PreRunE != nil {
			err := parent.PreRunE(parent, args)
			if err != nil {
				return err
			}
		} else {
			ParentPreRunE(parent, args)
		}
	}

	return nil
}
