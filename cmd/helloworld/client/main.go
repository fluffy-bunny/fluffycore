package main

import (
	context "context"
	"flag"
	"fmt"
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
	callUnarySayHello(rgc, "hello world")
}

func callUnarySayHello(client proto_helloworld.GreeterClient, message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.SayHello(ctx, &proto_helloworld.HelloRequest{Name: message})
	if err != nil {
		log.Fatal().Msgf("client.UnaryEcho(_) = _, %v: ", err)
	}
	fmt.Println("SayHello: ", resp.Message)
}
