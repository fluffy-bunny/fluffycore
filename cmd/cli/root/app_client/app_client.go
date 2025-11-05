package app_client

import (
	cobra_utils "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/cobra_utils"
	"github.com/fluffy-bunny/fluffycore/cmd/cli/root/app_client/helloworld"
	cobra "github.com/spf13/cobra"
)

const use = "app_client"

var printer = cobra_utils.NewPrinter()

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
		PreRunE:           cobra_utils.ParentPreRunE,
		Run: func(cmd *cobra.Command, args []string) {
			printer.EnableColors = true

		},
	}
	helloworld.Init(command)

	parentCmd.AddCommand(command)
}
