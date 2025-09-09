package GreeterServiceNATSMicroClientAccessor

import (
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contract_Greeter "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/contracts/Greeter"
	fluffycore_nats_client "github.com/fluffy-bunny/fluffycore/nats/client"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	nats "github.com/nats-io/nats.go"
)

type (
	service struct {
		conn          *nats.Conn
		greeterClient proto_helloworld.GreeterClient
		mutex         sync.Mutex
	}
)

var stemService = (*service)(nil)

var _ contract_Greeter.IGreeterNATSMicroClientAccessor = stemService

func (s *service) Ctor(conn *nats.Conn) (contract_Greeter.IGreeterNATSMicroClientAccessor, error) {
	return &service{conn: conn}, nil
}

func AddSingletonGreeterServiceNATSMicroClientAccessor(cb di.ContainerBuilder) {
	di.AddSingleton[contract_Greeter.IGreeterNATSMicroClientAccessor](cb, stemService.Ctor)
}

func (s *service) GetGreeterNATSMicroClient() proto_helloworld.GreeterClient {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.greeterClient == nil {
		greeterClient, err := proto_helloworld.NewGreeterNATSMicroClient(
			fluffycore_nats_client.WithNATSClientConn(s.conn),
		)
		if err != nil {
			return nil
		}
		s.greeterClient = greeterClient
	}
	return s.greeterClient
}
