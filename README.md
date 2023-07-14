# fluffycore

## Build the proto

[grpc.io](https://grpc.io/docs/languages/go/basics/)  
[Transcode Ref](https://grpc-ecosystem.github.io/grpc-gateway/docs/tutorials/introduction/)  
[custom protoc plugin](https://rotemtam.com/2021/03/22/creating-a-protoc-plugin-to-gen-go-code/)  

```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

```powershell
go mod tidy
cd proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative helloworld/helloworld.proto

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --go-fluffycore-di_out=.  --go-fluffycore-di_opt=paths=source_relative .\proto\helloworld\helloworld.proto 

```

### grpc Only

```powershell
go mod tidy
go build .\protoc-gen-go-fluffycore-di\cmd\protoc-gen-go-fluffycore-di\
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --go-fluffycore-di_out=.  --go-fluffycore-di_opt=paths=source_relative ./proto/helloworld/helloworld.proto 

```

### grpc and gateway

```powershell
go mod tidy
go build .\protoc-gen-go-fluffycore-di\cmd\protoc-gen-go-fluffycore-di\

protoc --go_out=. --go_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt paths=source_relative --go-grpc_out . --go-grpc_opt paths=source_relative --go-fluffycore-di_out .  --go-fluffycore-di_opt paths=source_relative,grpc_gateway=true  ./proto/helloworld/helloworld.proto  

```

## Why the custome protoc plugin?

The main reason is that we want a middleware to create a scoped container.  We want each request to instantiate the handler object within that container and subsequently only live for the life of the request.  Just like in the asp.net core world.  Since the proto defines the service we generate a shim for each call.  Imagine if the service had hundres of methods. We need that code generated.  

```go
// SayHello...
func (s *greeter2Server) SayHello(ctx context.Context, request *HelloRequest) (*HelloReply2, error) {
    requestContainer := dicontext.GetRequestContainer(ctx)
    downstreamService := di.Get[IGreeter2Server](requestContainer)
    return downstreamService.SayHello(ctx, request)
}
```

From the code we see that we pull the scoped container out of the context and request the downstream handler.  We make the exact same call on the handler.

## Streaming services

Streaming services are basically the same as a unary service.  The scoped context is that of the entire stream which could be VERY long lived.  A stream sends chunks of data in a single stream connection.  In an unary connection its just a single chunk, or request chunk and a response chunk.  So the scoped container covers a lot of chunks, and in that regard looses a bit of its value.  

Why do people stream anyway?

Because making a gazillion unary requests are wasteful.  
