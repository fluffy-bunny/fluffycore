package edge

import (
	"io"
	"math/rand"
	"time"

	di "github.com/dozm/di"
	endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	"github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
)

type (
	edgeServer struct {
		proto_helloworld.UnimplementedEdgeControlServiceServer
	}
)

func (s *edgeServer) Register(grpcServer *grpc.Server) {
	proto_helloworld.RegisterEdgeControlServiceServer(grpcServer, s)
}
func AddEdgeServer(cb di.ContainerBuilder) {
	di.AddSingleton[endpoint.IEndpointRegistration](cb, func() endpoint.IEndpointRegistration {
		return &edgeServer{}
	})
}

func (s *edgeServer) RequestPoints(request *proto_helloworld.PointsRequest, stream proto_helloworld.EdgeControlService_RequestPointsServer) error {
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
func (s *edgeServer) StreamPoints(stream proto_helloworld.EdgeControlService_StreamPointsServer) error {
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
