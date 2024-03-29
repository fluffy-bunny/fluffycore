/*
Copyright © 2023 Fluffy Bunny, LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	migrate "github.com/fluffy-bunny/fluffycore/cobracore/cmd/migrate"
	serve "github.com/fluffy-bunny/fluffycore/cobracore/cmd/serve"
	version "github.com/fluffy-bunny/fluffycore/cobracore/cmd/version"
	fluffycore_contracts_cobracore "github.com/fluffy-bunny/fluffycore/contracts/cobracore"
	fluffycore_contracts_runtime "github.com/fluffy-bunny/fluffycore/contracts/runtime"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

var cfgFile string

// SetVersion is the version of the application and is set from the main package
func SetVersion(v string) {
	version.SetVersion(v)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fluffycore",
	Short: "fluffy core cli",
	Long:  `fluffy core cli`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(startup fluffycore_contracts_runtime.IStartup, optionalCommands ...fluffycore_contracts_cobracore.ICommandInitializer) {
	serve.Startup = startup
	for _, command := range optionalCommands {
		command.Init(rootCmd)
	}
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobracore.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	version.Init(rootCmd)
	serve.Init(rootCmd)
	migrate.Init(rootCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cobracore" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("json")
		viper.SetConfigName(".cobracore")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
