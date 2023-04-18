package main

import (
	context "context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"time"

	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var addr = flag.String("addr", "localhost:50051", "the address to connect to")

func main() {
	flag.Parse()

	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Msgf("did not connect: %v", err)
	}
	defer conn.Close()

	// Make a echo client and send an RPC.
	rgc := proto_helloworld.NewGreeterClient(conn)
	streamer := proto_helloworld.NewEdgeControlServiceClient(conn)
	callUnarySayHello(rgc, "hello world")
	runRecordRoute(streamer)
	printPoints(streamer, &proto_helloworld.PointsRequest{
		Points: []string{"hello", "world"},
	})
}

func printPoints(client proto_helloworld.EdgeControlServiceClient, request *proto_helloworld.PointsRequest) {
	log.Info().Interface("request", request).Msgf("Looking for features within %v", request)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.RequestPoints(ctx, request)
	if err != nil {
		log.Fatal().Msgf("client.ListFeatures failed: %v", err)
	}
	for {
		point, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal().Msgf("client.ListFeatures failed: %v", err)
		}
		log.Info().Interface("point", point).Msg("point")

	}
}

func callUnarySayHello(client proto_helloworld.GreeterClient, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.SayHello(ctx, &proto_helloworld.HelloRequest{Name: message})
	if err != nil {
		log.Error().Msgf("client.UnaryEcho(_) = _, %v: ", err)
	} else {
		fmt.Println("SayHello: ", resp.Message)

	}
}
func randomPoint(r *rand.Rand) *proto_helloworld.Point {
	lat := (r.Int31n(180) - 90) * 1e7
	long := (r.Int31n(360) - 180) * 1e7
	return &proto_helloworld.Point{Latitude: lat, Longitude: long}
}

func runRecordRoute(client proto_helloworld.EdgeControlServiceClient) {
	// Create a random number of random points
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pointCount := int(r.Int31n(100)) + 2 // Traverse at least two points
	var points []*proto_helloworld.Point
	for i := 0; i < pointCount; i++ {
		points = append(points, randomPoint(r))
	}
	log.Info().Msgf("Traversing %d points.", len(points))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.StreamPoints(ctx)
	if err != nil {
		log.Fatal().Msgf("client.RecordRoute failed: %v", err)
	}
	for _, point := range points {
		if err := stream.Send(point); err != nil {
			log.Fatal().Msgf("client.RecordRoute: stream.Send(%v) failed: %v", point, err)
		}
	}
	reply, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal().Msgf("client.RecordRoute failed: %v", err)
	}
	log.Printf("Route summary: %v", reply)
}
