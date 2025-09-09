package greeter

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	cobra_utils "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/cobra_utils"
	services_GreeterServiceNATSMicroClientAccessor "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/services/GreeterServiceNATSMicroClientAccessor"
	internal_shared "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/shared"
	"github.com/fluffy-bunny/fluffycore/cmd/cli/root/nats/greeter/SayHelloAuth"
	cobra "github.com/spf13/cobra"
)

const use = "greeter"

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

			//fmt.Println(Version)
			return nil
		},
	}
	parentCmd.AddCommand(command)
	SayHelloAuth.Init(command)
}
func PreRunE(cmd *cobra.Command, args []string) error {
	internal_shared.AddServices(func(builder di.ContainerBuilder) {
		services_GreeterServiceNATSMicroClientAccessor.AddSingletonGreeterServiceNATSMicroClientAccessor(builder)
	})
	return cobra_utils.ParentPreRunE(cmd, args)
}
