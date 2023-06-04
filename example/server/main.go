package main

import (
	fluffycore_cobracore_cmd "github.com/fluffy-bunny/fluffycore/cobracore/cmd"
	internal_runtime "github.com/fluffy-bunny/fluffycore/example/internal/runtime"
	internal_version "github.com/fluffy-bunny/fluffycore/example/internal/version"
)

func main() {
	startup := internal_runtime.NewStartup()
	fluffycore_cobracore_cmd.SetVersion(internal_version.Version())
	fluffycore_cobracore_cmd.Execute(startup)
}
