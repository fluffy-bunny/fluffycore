package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal"
	"github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/startup"
	"github.com/fluffy-bunny/fluffycore/echo/runtime"
	"github.com/rs/zerolog/log"
)

var version = "Development"

func main() {
	processDirectory, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	internal.RootFolder = processDirectory
	fmt.Println("Version:" + version)
	DumpPath("./")
	r := runtime.New(startup.NewStartup())
	err = r.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run the application")
	}
}
