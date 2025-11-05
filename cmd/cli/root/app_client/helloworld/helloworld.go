package helloworld

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	cobra_utils "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/cobra_utils"
	internal_shared "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/shared"
	fluffycore_contracts_tokensource "github.com/fluffy-bunny/fluffycore/contracts/tokensource"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	cobra "github.com/spf13/cobra"
)

const use = "helloworld"

var printer = cobra_utils.NewPrinter()

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
		PreRunE:           PreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer.EnableColors = true
			ctx := internal_shared.GetContext()
			container := internal_shared.GetContainer()
			appTokenSource, err := di.TryGet[fluffycore_contracts_tokensource.IAppTokenSource](container)
			if err != nil {
				return err
			}
			tokenSource, err := appTokenSource.GetTokenSource()
			if err != nil {
				return err
			}
			token, err := tokenSource.Token()
			if err != nil {
				return err
			}
			printer.Printf(cobra_utils.Blue, "Access Token: %s\n", token.AccessToken)

			appGreeterClientAccessor, err := di.TryGet[proto_helloworld.IAppGreeterClientAccessor](container)
			if err != nil {
				return err
			}
			appGreeterClient, err := appGreeterClientAccessor.GetClient()
			if err != nil {
				return err
			}
			helloReply, err := appGreeterClient.SayHello(ctx,
				&proto_helloworld.HelloRequest{
					Name: "FluffyCore",
				})
			if err != nil {
				return err
			}
			printer.Printf(cobra_utils.Blue, "Greeting: %s\n", helloReply.Message)

			return nil
		},
	}
	parentCmd.AddCommand(command)
}
func PreRunE(cmd *cobra.Command, args []string) error {
	internal_shared.AddServices(func(builder di.ContainerBuilder) {
		proto_helloworld.AddSingletonIAppGreeterClientAccessor(builder,
			&proto_helloworld.AppGreeterClientAccessorConfig{
				Url: "localhost:50051",
			})

	})
	return cobra_utils.ParentPreRunE(cmd, args)
}
