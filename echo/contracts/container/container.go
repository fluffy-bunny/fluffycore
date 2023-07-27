package container

import (
	di "github.com/fluffy-bunny/fluffy-dozm-di"
)

type (
	ContainerAccessor func() di.Container
)
