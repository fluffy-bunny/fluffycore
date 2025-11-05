/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package root

import (
	"fmt"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	shared "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/shared"
	"github.com/fluffy-bunny/fluffycore/cmd/cli/root/app_client"
	"github.com/fluffy-bunny/fluffycore/cmd/cli/root/nats"
	"github.com/fluffy-bunny/fluffycore/cmd/cli/root/version"
	fluffycore_contracts_GRPCClientFactory "github.com/fluffy-bunny/fluffycore/contracts/GRPCClientFactory"
	fluffycore_contracts_tokensource "github.com/fluffy-bunny/fluffycore/contracts/tokensource"
	fluffycore_services_GRPCClientFactory "github.com/fluffy-bunny/fluffycore/services/GRPCClientFactory"
	fluffycore_services_tokensource_client_credentials "github.com/fluffy-bunny/fluffycore/services/tokensource/client_credentials"

	//GRPCClientFactory
	godotenv "github.com/joho/godotenv"

	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const (
	prettyLogFlagName    = "pretty-log"
	prettyLogEnvVariable = "PRETTY_LOG"

	logLevelFlagName    = "log-level"
	logLevelEnvVariable = "LOG_LEVEL"
)

var (
	prettyLog bool
	logLevel  string
	env_path  string
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(cmd *cobra.Command) {
	cobra.CheckErr(cmd.Execute())
}

// ExecuteE adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteE(cmd *cobra.Command) error {
	return cmd.Execute()
}

// InitRootCmd initializes the root command
func InitRootCmd() *cobra.Command {
	// command represents the base command when called without any subcommands
	var command = &cobra.Command{
		Use:   "cli",
		Short: "A gRPC Core Template CLI Tool",
		Long:  ``,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// everyone should get access to the envs
			err := godotenv.Load(env_path)
			if err != nil {
				return err
			}
			// the builder is setup here.
			// register your services in the PreRunE
			builder := di.Builder()
			shared.SetBuilder(builder)
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {

			appTokenSourceConfig := &fluffycore_contracts_tokensource.AppTokenSourceConfig{
				ClientID:     shared.OAuth2.ClientID,
				ClientSecret: shared.OAuth2.ClientSecret,
				TokenURL:     shared.OAuth2.TokenEndepoint,
				Scopes:       []string{},
			}

			shared.AddServices(func(builder di.ContainerBuilder) {
				fluffycore_services_tokensource_client_credentials.AddSingletonIAppTokenSource(builder, appTokenSourceConfig)
				fluffycore_services_GRPCClientFactory.AddSingletonIGRPCClientFactory(builder, &fluffycore_contracts_GRPCClientFactory.GRPCClientConfig{
					OTELTracingEnabled:    true,
					DataDogTracingEnabled: false,
				})
			})
			shared.BuildContainer()
			return nil
		},
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
	}
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kafkaCLI.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	var flagName string
	var sDefault string
	flagName = logLevelFlagName

	flagName = "env_path"
	sDefault = "local.env"
	command.PersistentFlags().StringVar(&env_path, flagName, sDefault, fmt.Sprintf("i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = prettyLogFlagName
	command.PersistentFlags().BoolVar(&prettyLog, flagName, false, fmt.Sprintf("i.e. --%s=true", flagName))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "oauth2-client-id"
	sDefault = "client1"
	command.PersistentFlags().StringVar(&shared.OAuth2.ClientID, flagName, sDefault, fmt.Sprintf("[required] i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "oauth2-client-secret"
	sDefault = "secret"
	command.PersistentFlags().StringVar(&shared.OAuth2.ClientSecret, flagName, sDefault, fmt.Sprintf("[required] i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "oauth2-token-endpoint"
	sDefault = "http://localhost:50053/oauth/token"
	command.PersistentFlags().StringVar(&shared.OAuth2.TokenEndepoint, flagName, sDefault, fmt.Sprintf("[required] i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "env-file-path"
	sDefault = ".env"
	command.PersistentFlags().StringVar(&shared.EnvFilePath, flagName, sDefault, fmt.Sprintf("  i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	version.Init(command)
	nats.Init(command)
	app_client.Init(command)
	return command
}
