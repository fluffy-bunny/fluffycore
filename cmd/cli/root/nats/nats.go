package nats

import (
	"fmt"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	cobra_utils "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/cobra_utils"
	shared "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/shared"
	"github.com/fluffy-bunny/fluffycore/cmd/cli/root/nats/greeter"
	fluffycore_nats_token "github.com/fluffy-bunny/fluffycore/nats/nats_token"
	nats "github.com/nats-io/nats.go"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "nats"

var printer = cobra_utils.NewPrinter()

var (
	natsUrl          string
	natsClientId     string
	natsClientSecret string
)

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
		PreRunE:           PreRunE,
		Run: func(cmd *cobra.Command, args []string) {
			printer.EnableColors = true

			//fmt.Println(Version)
		},
	}

	var flagName string
	var sDefault string

	flagName = "natsUrl"
	sDefault = "nats://127.0.0.1:4222"
	command.PersistentFlags().StringVar(&natsUrl, flagName, sDefault, fmt.Sprintf("i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "natsClientId"
	sDefault = "nats-micro-god"
	command.PersistentFlags().StringVar(&natsClientId, flagName, sDefault, fmt.Sprintf("i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "natsClientSecret"
	sDefault = "secret"
	command.PersistentFlags().StringVar(&natsClientSecret, flagName, sDefault, fmt.Sprintf("i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)
	greeter.Init(command)

	/*
			"natsMicroConfig": {
			"natsUrl": "nats://127.0.0.1:4222",
			"clientId": "nats-micro-god",
			"clientSecret": "secret",
			"timeoutDuration": "5s"
		}
	*/
}

func PreRunE(cmd *cobra.Command, args []string) error {
	nc, err := fluffycore_nats_token.CreateNatsConnectionWithClientCredentials(
		&fluffycore_nats_token.NATSConnectTokenClientCredentialsRequest{
			NATSUrl:      natsUrl,
			ClientID:     natsClientId,
			ClientSecret: natsClientSecret,
		},
	)
	if err != nil {
		return err
	}
	//fmt.Println(utils.PrettyJSON(zendeskPasswordCreds))
	shared.AddServices(func(builder di.ContainerBuilder) {
		di.AddInstance[*nats.Conn](builder, nc)
	})

	return cobra_utils.ParentPreRunE(cmd, args)
}
