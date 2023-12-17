package cobracore

import (
	"github.com/spf13/cobra"
)

type (
	ICommandInitializer interface {
		Init(rootCmd *cobra.Command)
	}
)
