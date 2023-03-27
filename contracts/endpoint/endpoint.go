package endpoint

import "google.golang.org/grpc"

type (
	IEndpointRegistration interface {
		Register(s *grpc.Server)
	}
)
