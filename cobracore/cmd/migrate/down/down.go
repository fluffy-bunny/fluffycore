package down

import (
	"github.com/fluffy-bunny/fluffycore/cobracore/cmd/migrate/utils"
	"github.com/spf13/cobra"
)

var command = &cobra.Command{
	Use:               "down",
	Short:             "migrates the db down",
	PersistentPreRunE: utils.UpPersistentPreRunE,
	RunE: func(cmd *cobra.Command, args []string) error {
		return utils.Migrate(utils.MigrateDown)
	},
}

func Init(rootCmd *cobra.Command) {
	rootCmd.AddCommand(command)
}
