package main

import (
	"fmt"

	fluffycore_cobracore_cmd "github.com/fluffy-bunny/fluffycore/cobracore/cmd"
	internal_runtime "github.com/fluffy-bunny/fluffycore/example/internal/runtime"
)

func main() {
	fmt.Println("Hello, playground")
	startup := internal_runtime.NewStartup()
	fluffycore_cobracore_cmd.Execute(startup)
}
