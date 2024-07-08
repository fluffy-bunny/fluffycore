# fluffycore

## Build the proto

[grpc.io](https://grpc.io/docs/languages/go/basics/)  
[Transcode Ref](https://grpc-ecosystem.github.io/grpc-gateway/docs/tutorials/introduction/)  
[custom protoc plugin](https://rotemtam.com/2021/03/22/creating-a-protoc-plugin-to-gen-go-code/)

```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
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

Note: I had to run bash on windows so I could pass `./api/proto/**/*.proto`

```bash
go mod tidy
go build .\protoc-gen-go-fluffycore-di\cmd\protoc-gen-go-fluffycore-di\

protoc --go_out=. --go_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt paths=source_relative --go-grpc_out . --go-grpc_opt paths=source_relative --go-fluffycore-di_out .  --go-fluffycore-di_opt paths=source_relative,grpc_gateway=true  ./proto/helloworld/helloworld.proto

protoc --go_out=. --go_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt paths=source_relative --go-grpc_out . --go-grpc_opt paths=source_relative --go-fluffycore-di_out .  --go-fluffycore-di_opt paths=source_relative,grpc_gateway=true  ./proto/someservice/someservice.proto

protoc  --go_out=. --go_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt logtostderr=true --grpc-gateway_opt paths=source_relative --openapiv2_out=allow_merge=true,merge_file_name=myawesomeapi:./proto/swagger --go-grpc_out . --go-grpc_opt paths=source_relative --go-fluffycore-di_out .  --go-fluffycore-di_opt paths=source_relative,grpc_gateway=true  ./proto/**/*.proto



protoc -I./api/proto \
    -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway \
    --grpc-gateway_out=logtostderr=true:./pkg \
    --swagger_out=allow_merge=true,merge_file_name=myawesomeapi:./api/swagger \
    --go_out=plugins=grpc:pkg ./api/proto/**/*.proto

```

## Why the custome protoc plugin?

The main reason is that we want a middleware to create a scoped container. We want each request to instantiate the handler object within that container and subsequently only live for the life of the request. Just like in the asp.net core world. Since the proto defines the service we generate a shim for each call. Imagine if the service had hundres of methods. We need that code generated.

```go
// SayHello...
func (s *greeter2Server) SayHello(ctx context.Context, request *HelloRequest) (*HelloReply2, error) {
    requestContainer := dicontext.GetRequestContainer(ctx)
    downstreamService := di.Get[IGreeter2Server](requestContainer)
    return downstreamService.SayHello(ctx, request)
}
```

From the code we see that we pull the scoped container out of the context and request the downstream handler. We make the exact same call on the handler.

## Streaming services

Streaming services are basically the same as a unary service. The scoped context is that of the entire stream which could be VERY long lived. A stream sends chunks of data in a single stream connection. In an unary connection its just a single chunk, or request chunk and a response chunk. So the scoped container covers a lot of chunks, and in that regard looses a bit of its value.

Why do people stream anyway?

Because making a gazillion unary requests are wasteful.

## GRPC Gateway

[customizing_openapi_output](https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/customizing_openapi_output/)

[7-tips-when-working-with-grpc-gateways-swagger-support](https://medium.com/golang-diary/7-tips-when-working-with-grpc-gateways-swagger-support-afa0c2d671d8)

## Docker

```bash
 docker build --file .\build\Dockerfile . --tag fluffycore.example
```

## OpenTelemetry

[intro-to-o11y-go](https://github.com/honeycombio/intro-to-o11y-go)  
[uptrace](https://github.com/uptrace/opentelemetry-go-extra/tree/main/example/grpc)  
[exploring-the-opentelemetry-client-library-for-go](https://medium.com/@tennis.akari.abcdefg/exploring-the-opentelemetry-client-library-for-go-5b75c92a74a5)
