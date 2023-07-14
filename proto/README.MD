# ProtoC  

## generators

[Transcode Ref](https://grpc-ecosystem.github.io/grpc-gateway/docs/tutorials/introduction/)  

```bash
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## dependencies

Do this in the root of the project

```bash
git clone https://github.com/googleapis/googleapis.git
# copy to google/api is so that the linter can find the files in visual studio. 
# leave the googleapis folder in place for the protoc to find the files
md ./google/api
cp ./googleapis/google/api/annotations.proto ./google/api/annotations.proto
cp ./googleapis/google/api/field_behavior.proto ./google/api/field_behavior.proto
cp ./googleapis/google/api/http.proto ./google/api/http.proto
cp ./googleapis/google/api/httpbody.proto ./google/api/httpbody.proto
```

## Individual outputs  

go mod vendor

```powershell

cd ./proto

protoc --go_out=. --go_opt=paths=source_relative --proto_path=../googleapis --proto_path=.  --go-grpc_out=. --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative  --go-grpc_opt=paths=source_relative helloworld/helloworld.proto

```
