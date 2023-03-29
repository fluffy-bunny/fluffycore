package main

import (
	"fmt"

	fluffycore_cobracore_cmd "github.com/fluffy-bunny/fluffycore/cobracore/cmd"
)

func main() {
	fmt.Println("Hello, playground")
	startup := NewStartup()
	fluffycore_cobracore_cmd.Execute(startup)
}
