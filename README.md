# fluffy-core

## Build the proto

```powershell
go mod tidy
cd proto
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative helloworld/helloworld.proto

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --go-fluffycore-di_out=.  --go-fluffycore-di_opt=paths=source_relative .\proto\helloworld\helloworld.proto 

```

```powershell
go mod tidy
go build .\protoc-gen-go-fluffycore-di\cmd\protoc-gen-go-fluffycore-di\
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --go-fluffycore-di_out=.  --go-fluffycore-di_opt=paths=source_relative ./proto/helloworld/helloworld.proto 

```
