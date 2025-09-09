/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"os"

	shared "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/shared"
	"github.com/fluffy-bunny/fluffycore/cmd/cli/root"
	"github.com/rs/zerolog"
)

func main() {

	ctx := context.Background()
	log := zerolog.New(os.Stdout).With().Caller().Timestamp().Logger()
	ctx = log.WithContext(ctx)
	shared.SetContext(ctx)

	rootCommand := root.InitRootCmd()
	err := root.ExecuteE(rootCommand)
	if err != nil {
		log.Error().Err(err).Msg("error executing command")
	}
}
