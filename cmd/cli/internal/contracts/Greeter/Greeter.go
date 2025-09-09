package Greeter

import (
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
)

type (
	IGreeterNATSMicroClientAccessor interface {
		GetGreeterNATSMicroClient() proto_helloworld.GreeterClient
	}
)
