// Code generated by protoc-gen-grpc-gateway. DO NOT EDIT.
// source: proto/someservice/someservice.proto

/*
Package someservice is a reverse proxy.

It translates gRPC into RESTful JSON APIs.
*/
package someservice

import (
	"context"
	"io"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Suppress "imported and not used" errors
var _ codes.Code
var _ io.Reader
var _ status.Status
var _ = runtime.String
var _ = utilities.NewDoubleArray
var _ = metadata.Join

func request_SomeService_SayHello_0(ctx context.Context, marshaler runtime.Marshaler, client SomeServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq HelloRequest
	var metadata runtime.ServerMetadata

	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	msg, err := client.SayHello(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func local_request_SomeService_SayHello_0(ctx context.Context, marshaler runtime.Marshaler, server SomeServiceServer, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq HelloRequest
	var metadata runtime.ServerMetadata

	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	msg, err := server.SayHello(ctx, &protoReq)
	return msg, metadata, err

}

func request_SomeService2_SayHello_0(ctx context.Context, marshaler runtime.Marshaler, client SomeService2Client, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq HelloRequest
	var metadata runtime.ServerMetadata

	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	msg, err := client.SayHello(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func local_request_SomeService2_SayHello_0(ctx context.Context, marshaler runtime.Marshaler, server SomeService2Server, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq HelloRequest
	var metadata runtime.ServerMetadata

	if err := marshaler.NewDecoder(req.Body).Decode(&protoReq); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	msg, err := server.SayHello(ctx, &protoReq)
	return msg, metadata, err

}

// RegisterSomeServiceHandlerServer registers the http handlers for service SomeService to "mux".
// UnaryRPC     :call SomeServiceServer directly.
// StreamingRPC :currently unsupported pending https://github.com/grpc/grpc-go/issues/906.
// Note that using this registration option will cause many gRPC library features to stop working. Consider using RegisterSomeServiceHandlerFromEndpoint instead.
func RegisterSomeServiceHandlerServer(ctx context.Context, mux *runtime.ServeMux, server SomeServiceServer) error {

	mux.Handle("POST", pattern_SomeService_SayHello_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		var err error
		var annotatedContext context.Context
		annotatedContext, err = runtime.AnnotateIncomingContext(ctx, mux, req, "/someservice.SomeService/SayHello", runtime.WithHTTPPathPattern("/v1/someservice/echo"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_SomeService_SayHello_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_SomeService_SayHello_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

// RegisterSomeService2HandlerServer registers the http handlers for service SomeService2 to "mux".
// UnaryRPC     :call SomeService2Server directly.
// StreamingRPC :currently unsupported pending https://github.com/grpc/grpc-go/issues/906.
// Note that using this registration option will cause many gRPC library features to stop working. Consider using RegisterSomeService2HandlerFromEndpoint instead.
func RegisterSomeService2HandlerServer(ctx context.Context, mux *runtime.ServeMux, server SomeService2Server) error {

	mux.Handle("POST", pattern_SomeService2_SayHello_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		var stream runtime.ServerTransportStream
		ctx = grpc.NewContextWithServerTransportStream(ctx, &stream)
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		var err error
		var annotatedContext context.Context
		annotatedContext, err = runtime.AnnotateIncomingContext(ctx, mux, req, "/someservice.SomeService2/SayHello", runtime.WithHTTPPathPattern("/v2/someservice/echo"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_SomeService2_SayHello_0(annotatedContext, inboundMarshaler, server, req, pathParams)
		md.HeaderMD, md.TrailerMD = metadata.Join(md.HeaderMD, stream.Header()), metadata.Join(md.TrailerMD, stream.Trailer())
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_SomeService2_SayHello_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

// RegisterSomeServiceHandlerFromEndpoint is same as RegisterSomeServiceHandler but
// automatically dials to "endpoint" and closes the connection when "ctx" gets done.
func RegisterSomeServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error) {
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				grpclog.Errorf("Failed to close conn to %s: %v", endpoint, cerr)
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				grpclog.Errorf("Failed to close conn to %s: %v", endpoint, cerr)
			}
		}()
	}()

	return RegisterSomeServiceHandler(ctx, mux, conn)
}

// RegisterSomeServiceHandler registers the http handlers for service SomeService to "mux".
// The handlers forward requests to the grpc endpoint over "conn".
func RegisterSomeServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return RegisterSomeServiceHandlerClient(ctx, mux, NewSomeServiceClient(conn))
}

// RegisterSomeServiceHandlerClient registers the http handlers for service SomeService
// to "mux". The handlers forward requests to the grpc endpoint over the given implementation of "SomeServiceClient".
// Note: the gRPC framework executes interceptors within the gRPC handler. If the passed in "SomeServiceClient"
// doesn't go through the normal gRPC flow (creating a gRPC client etc.) then it will be up to the passed in
// "SomeServiceClient" to call the correct interceptors.
func RegisterSomeServiceHandlerClient(ctx context.Context, mux *runtime.ServeMux, client SomeServiceClient) error {

	mux.Handle("POST", pattern_SomeService_SayHello_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		var err error
		var annotatedContext context.Context
		annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/someservice.SomeService/SayHello", runtime.WithHTTPPathPattern("/v1/someservice/echo"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_SomeService_SayHello_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_SomeService_SayHello_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

var (
	pattern_SomeService_SayHello_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2}, []string{"v1", "someservice", "echo"}, ""))
)

var (
	forward_SomeService_SayHello_0 = runtime.ForwardResponseMessage
)

// RegisterSomeService2HandlerFromEndpoint is same as RegisterSomeService2Handler but
// automatically dials to "endpoint" and closes the connection when "ctx" gets done.
func RegisterSomeService2HandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error) {
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				grpclog.Errorf("Failed to close conn to %s: %v", endpoint, cerr)
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				grpclog.Errorf("Failed to close conn to %s: %v", endpoint, cerr)
			}
		}()
	}()

	return RegisterSomeService2Handler(ctx, mux, conn)
}

// RegisterSomeService2Handler registers the http handlers for service SomeService2 to "mux".
// The handlers forward requests to the grpc endpoint over "conn".
func RegisterSomeService2Handler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return RegisterSomeService2HandlerClient(ctx, mux, NewSomeService2Client(conn))
}

// RegisterSomeService2HandlerClient registers the http handlers for service SomeService2
// to "mux". The handlers forward requests to the grpc endpoint over the given implementation of "SomeService2Client".
// Note: the gRPC framework executes interceptors within the gRPC handler. If the passed in "SomeService2Client"
// doesn't go through the normal gRPC flow (creating a gRPC client etc.) then it will be up to the passed in
// "SomeService2Client" to call the correct interceptors.
func RegisterSomeService2HandlerClient(ctx context.Context, mux *runtime.ServeMux, client SomeService2Client) error {

	mux.Handle("POST", pattern_SomeService2_SayHello_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		var err error
		var annotatedContext context.Context
		annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/someservice.SomeService2/SayHello", runtime.WithHTTPPathPattern("/v2/someservice/echo"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := request_SomeService2_SayHello_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_SomeService2_SayHello_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

var (
	pattern_SomeService2_SayHello_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2}, []string{"v2", "someservice", "echo"}, ""))
)

var (
	forward_SomeService2_SayHello_0 = runtime.ForwardResponseMessage
)
