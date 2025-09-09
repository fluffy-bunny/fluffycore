package version

import (
	cobra_utils "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/cobra_utils"
	cobra "github.com/spf13/cobra"
)

// Version global
var Version string

func init() {
	SetVersion("0.0.0")
}

// SetVersion ...
func SetVersion(version string) {
	Version = version
}

const use = "version"

var printer = cobra_utils.NewPrinter()

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             "Print the version number of the app",
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
		Run: func(cmd *cobra.Command, args []string) {
			printer.EnableColors = true
			printer.Infof("version: %s", Version)

			//fmt.Println(Version)
		},
	}
	parentCmd.AddCommand(command)
}
