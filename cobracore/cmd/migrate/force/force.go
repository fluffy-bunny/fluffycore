package force

import (
	"github.com/fluffy-bunny/fluffycore/cobracore/cmd/migrate/utils"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var command = &cobra.Command{
	Use:               "force",
	Short:             "Force sets a migration version. It does not check any currently active version in database. It resets the dirty state to false.",
	PersistentPreRunE: utils.UpPersistentPreRunE,
	RunE: func(cmd *cobra.Command, args []string) error {
		return utils.Migrate(utils.MigrateForce)
	},
}
var version int

func Init(rootCmd *cobra.Command) {
	rootCmd.AddCommand(command)
	// a migration version
	command.Flags().IntVar(&version, "version", -1, "[required] a migration version")
	command.MarkFlagRequired("version")
	// version is registered on Flags() above, not PersistentFlags() - must
	// look it up from the same set, or Lookup returns nil, BindPFlag's
	// (ignored) error fires, and viper.GetInt("version") silently falls
	// back to its zero value instead of whatever --version was passed.
	// Confirmed live: `migrate force --version 16` was actually calling
	// m.Force(0).
	if err := viper.BindPFlag("version", command.Flags().Lookup("version")); err != nil {
		log.Error().Err(err).Msg("failed to bind --version flag to viper")
	}
	viper.BindEnv("version", "FORCE_VERSION")
}
