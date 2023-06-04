package mystream

import (
	"io"
	"math/rand"
	"time"

	di "github.com/dozm/di"
	contracts_config "github.com/fluffy-bunny/fluffycore/example/internal/contracts/config"
	fluffycore_contracts_somedisposable "github.com/fluffy-bunny/fluffycore/example/internal/contracts/somedisposable"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	"github.com/rs/zerolog/log"
)

type (
	service struct {
		proto_helloworld.UnimplementedMyStreamServiceServer
		config               *contracts_config.Config
		scopedSomeDisposable fluffycore_contracts_somedisposable.IScopedSomeDisposable
	}
)

func AddMyStreamService(builder di.ContainerBuilder) {
	proto_helloworld.AddMyStreamServiceServer[proto_helloworld.IMyStreamServiceServer](builder,
		func(config *contracts_config.Config, scopedSomeDisposable fluffycore_contracts_somedisposable.IScopedSomeDisposable) proto_helloworld.IMyStreamServiceServer {
			return &service{
				config:               config,
				scopedSomeDisposable: scopedSomeDisposable,
			}
		})
}

func (s *service) RequestPoints(request *proto_helloworld.PointsRequest, stream proto_helloworld.MyStreamService_RequestPointsServer) error {

	numPoints := len(request.Points)
	for i := 0; i < numPoints; i++ {
		for j := 0; j < 10; j++ {
			err := stream.Send(&proto_helloworld.Point{
				Latitude:  rand.Int31n(100),
				Longitude: rand.Int31n(100),
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
func (s *service) StreamPoints(stream proto_helloworld.MyStreamService_StreamPointsServer) error {
	var pointCount int32
	startTime := time.Now()
	for {
		point, err := stream.Recv()
		log.Info().Interface("point", point).Msg("StreamPoints")
		if err == io.EOF {
			endTime := time.Now()
			return stream.SendAndClose(&proto_helloworld.RouteSummary{
				PointCount: pointCount,

				ElapsedTime: int32(endTime.Sub(startTime).Seconds()),
			})
		}
		if err != nil {
			return err
		}
		pointCount++

	}
}
