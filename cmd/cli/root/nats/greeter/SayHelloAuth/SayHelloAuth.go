package SayHelloAuth

import (
	"fmt"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	cobra_utils "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/cobra_utils"
	contract_greeter "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/contracts/Greeter"
	internal_shared "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/shared"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	cobra "github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/metadata"
)

const use = "sayhelloauth"

var printer = cobra_utils.NewPrinter()
var (
	vinInput string
)

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
		PreRunE:           cobra_utils.ParentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer.EnableColors = true
			ctx := internal_shared.GetContext()
			container := internal_shared.GetContainer()

			tokenSources, err := di.TryGet[oauth2.TokenSource](container)
			if err != nil {
				return err
			}
			serviceNATSMicroClientAccessor, err := di.TryGet[contract_greeter.IGreeterNATSMicroClientAccessor](container)
			if err != nil {
				return err
			}
			// add token to grpc metadata
			token, err := tokenSources.Token()
			if err != nil {
				return err
			}

			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token.AccessToken)

			greeterClient := serviceNATSMicroClientAccessor.GetGreeterNATSMicroClient()
			helloReply, err := greeterClient.SayHelloAuth(ctx,
				&proto_helloworld.HelloRequest{
					Name: "Toyota",
				})

			if err != nil {
				return err
			}
			fmt.Println(internal_shared.PrettyJSON(helloReply))
			return nil
		},
	}

	var flagName string
	var sDefault string

	flagName = "vin"
	sDefault = "1234567890"
	command.PersistentFlags().StringVar(&vinInput, flagName, sDefault, fmt.Sprintf("i.e. --%s=%s", flagName, sDefault))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)
}
