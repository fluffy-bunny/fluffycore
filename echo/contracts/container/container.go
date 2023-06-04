package container

import (
	di "github.com/dozm/di"
)

type (
	ContainerAccessor func() di.Container
)
