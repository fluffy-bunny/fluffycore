package nats_micro_service

// Integration tests against a real NATS server. Skipped unless NATS_TEST_URL is set.
// Easiest: ./scripts/test-nats.sh  (docker compose up, go test ./nats/..., down), or by hand:
//
//	docker compose -f docker/docker-compose.nats.yml up -d --wait
//	NATS_TEST_URL=nats://localhost:4299 go test ./nats/...

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	nats_client "github.com/fluffy-bunny/fluffycore/nats/client"
	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	"github.com/stretchr/testify/require"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
	proto "google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func natsConn(t *testing.T) *nats.Conn {
	t.Helper()
	url := os.Getenv("NATS_TEST_URL")
	if url == "" {
		t.Skip("NATS_TEST_URL not set; skipping NATS integration test")
	}
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	return nc
}

func natsClient(t *testing.T, nc *nats.Conn, timeout time.Duration) *nats_client.NATSClient {
	t.Helper()
	c, err := nats_client.NewNATSClient(nats_client.WithNATSClientConn(nc), nats_client.WithTimeout(timeout))
	require.NoError(t, err)
	return c
}

// A real micro service: the error status and an all-defaults reply both survive the trip.
func TestNATS_ServiceStatusAndEmptyReply(t *testing.T) {
	nc := natsConn(t)
	svc, err := micro.AddService(nc, micro.Config{Name: "fluffytest", Version: "0.0.1"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Stop() })
	require.NoError(t, svc.AddEndpoint("missing", micro.HandlerFunc(func(req micro.Request) {
		respondError(req, grpcstatus.Error(codes.NotFound, "no such thing"))
	}), micro.WithEndpointSubject("fluffytest.missing")))
	require.NoError(t, svc.AddEndpoint("empty", micro.HandlerFunc(func(req micro.Request) {
		b, _ := proto.Marshal(&wrapperspb.StringValue{}) // all defaults: zero bytes
		_ = req.Respond(b)
	}), micro.WithEndpointSubject("fluffytest.empty")))

	client := natsClient(t, nc, 5*time.Second)
	ctx := context.Background()

	_, err = HandleNATSClientRequest(ctx, client, "fluffytest.missing", &wrapperspb.StringValue{}, &wrapperspb.StringValue{})
	require.Equal(t, codes.NotFound, grpcstatus.Code(err), "used to arrive as Internal: %v", err)
	require.Equal(t, "no such thing", grpcstatus.Convert(err).Message())

	resp, err := HandleNATSClientRequest(ctx, client, "fluffytest.empty", &wrapperspb.StringValue{}, &wrapperspb.StringValue{})
	require.NoError(t, err, "an all-defaults reply used to be rejected as empty")
	require.Equal(t, "", resp.GetValue())
}

// RequestWithContext used to ignore ctx and always wait the client timeout.
func TestNATS_RequestWithContextHonoursCancellation(t *testing.T) {
	nc := natsConn(t)
	sub, err := nc.Subscribe("fluffytest.slow", func(*nats.Msg) {}) // never replies
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	require.NoError(t, nc.Flush())

	client := natsClient(t, nc, 30*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = client.RequestWithContext(ctx, "fluffytest.slow", []byte("x"))
	require.Error(t, err)
	require.Less(t, time.Since(start), 5*time.Second, "must return when ctx ends, not after the 30s client timeout")
}

// The whole path over a real server: ReplaceTokens builds the subject, the endpoint's
// HandleRequest injects the token back into the request, the service answers or fails with a status.
func TestNATS_EndToEndParameterizedRoute(t *testing.T) {
	nc := natsConn(t)
	svc, err := micro.AddService(nc, micro.Config{Name: "fluffye2e", Version: "0.0.1"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Stop() })

	const parameterized = "fluffye2e.get.${name}"
	require.NoError(t, svc.AddEndpoint("get", micro.HandlerFunc(func(req micro.Request) {
		HandleRequest(req, "", &NATSMicroHandlerInfo{ParameterizedToken: parameterized},
			func(r *descriptorpb.FileDescriptorProto) error { return proto.Unmarshal(req.Data(), r) },
			func() (protoreflect.ProtoMessage, error) {
				m := &descriptorpb.FileDescriptorProto{}
				return m, proto.Unmarshal(req.Data(), m)
			},
			func(m protoreflect.ProtoMessage, r *descriptorpb.FileDescriptorProto) error {
				r.Name = m.(*descriptorpb.FileDescriptorProto).Name
				return nil
			},
			func(_ context.Context, r *descriptorpb.FileDescriptorProto) (*wrapperspb.StringValue, error) {
				switch r.GetName() {
				case "missing":
					return nil, grpcstatus.Error(codes.NotFound, "no such name")
				case "denied":
					return nil, grpcstatus.Error(codes.PermissionDenied, "not yours")
				case "boom":
					panic("handler blew up")
				}
				return wrapperspb.String("hello " + r.GetName()), nil
			})
	}), micro.WithEndpointSubject("fluffye2e.get.*")))

	client := natsClient(t, nc, 5*time.Second)
	call := func(name string) (*wrapperspb.StringValue, error) {
		req := &descriptorpb.FileDescriptorProto{Name: proto.String(name)}
		subject, err := ReplaceTokens(parameterized, req)
		if err != nil {
			return nil, err
		}
		return HandleNATSClientRequest(context.Background(), client, subject, req, &wrapperspb.StringValue{})
	}

	resp, err := call("acme")
	require.NoError(t, err)
	require.Equal(t, "hello acme", resp.GetValue(), "the subject token must arrive as the request field")

	_, err = call("missing")
	require.Equal(t, codes.NotFound, grpcstatus.Code(err))
	_, err = call("denied")
	require.Equal(t, codes.PermissionDenied, grpcstatus.Code(err))

	// A value that would change the subject never reaches the wire.
	for _, v := range []string{"a.b", "*", ">"} {
		_, err = call(v)
		require.Equal(t, codes.InvalidArgument, grpcstatus.Code(err), "value %q", v)
	}
}

// A panicking handler must still answer; otherwise the caller waits out its whole timeout.
func TestNATS_PanickingHandlerStillResponds(t *testing.T) {
	nc := natsConn(t)
	svc, err := micro.AddService(nc, micro.Config{Name: "fluffypanic", Version: "0.0.1"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Stop() })
	require.NoError(t, svc.AddEndpoint("boom", micro.HandlerFunc(func(req micro.Request) {
		HandleRequest(req, "", &NATSMicroHandlerInfo{ParameterizedToken: "fluffypanic.boom"},
			func(*wrapperspb.StringValue) error { return nil },
			func() (protoreflect.ProtoMessage, error) { return &wrapperspb.StringValue{}, nil },
			func(protoreflect.ProtoMessage, *wrapperspb.StringValue) error { return nil },
			func(context.Context, *wrapperspb.StringValue) (*wrapperspb.StringValue, error) { panic("boom") })
	}), micro.WithEndpointSubject("fluffypanic.boom")))

	client := natsClient(t, nc, 10*time.Second)
	start := time.Now()
	_, err = HandleNATSClientRequest(context.Background(), client, "fluffypanic.boom", &wrapperspb.StringValue{}, &wrapperspb.StringValue{})
	require.Equal(t, codes.Internal, grpcstatus.Code(err), "got %v", err)
	require.Less(t, time.Since(start), 3*time.Second, "must answer, not time out")
}

// Nobody is listening: that is Unavailable, not Unknown.
func TestNATS_NoRespondersIsUnavailable(t *testing.T) {
	nc := natsConn(t)
	client := natsClient(t, nc, 5*time.Second)
	_, err := HandleNATSClientRequest(context.Background(), client, "fluffytest.nobody.home", &wrapperspb.StringValue{}, &wrapperspb.StringValue{})
	require.Equal(t, codes.Unavailable, grpcstatus.Code(err), "got %v", err)
}

// The caller's context deadline reaches the caller as DeadlineExceeded.
func TestNATS_ContextDeadlineIsDeadlineExceeded(t *testing.T) {
	nc := natsConn(t)
	sub, err := nc.Subscribe("fluffytest.hang", func(*nats.Msg) {})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	require.NoError(t, nc.Flush())

	client := natsClient(t, nc, 30*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = HandleNATSClientRequest(ctx, client, "fluffytest.hang", &wrapperspb.StringValue{}, &wrapperspb.StringValue{})
	require.Equal(t, codes.DeadlineExceeded, grpcstatus.Code(err), "got %v", err)
}

// gRPC metadata set by the caller (auth, tracing, ...) must arrive in the service method's
// context, through the interceptor, the NATS headers and HandleRequest.
func TestNATS_InterceptorPropagatesMetadata(t *testing.T) {
	nc := natsConn(t)
	svc, err := micro.AddService(nc, micro.Config{Name: "fluffyhdr", Version: "0.0.1"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Stop() })
	require.NoError(t, svc.AddEndpoint("echo", micro.HandlerFunc(func(req micro.Request) {
		HandleRequest(req, "", &NATSMicroHandlerInfo{ParameterizedToken: "fluffyhdr.echo"},
			func(*wrapperspb.StringValue) error { return nil },
			func() (protoreflect.ProtoMessage, error) { return &wrapperspb.StringValue{}, nil },
			func(protoreflect.ProtoMessage, *wrapperspb.StringValue) error { return nil },
			func(ctx context.Context, _ *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
				// the service's context carries the caller's metadata (outgoing, for forwarding)
				md, _ := metadata.FromOutgoingContext(ctx)
				return wrapperspb.String(strings.Join(md.Get("authorization"), ",") + "|" + strings.Join(md.Get("x-request-id"), ",")), nil
			})
	}), micro.WithEndpointSubject("fluffyhdr.echo")))

	client := natsClient(t, nc, 5*time.Second)
	invoke := SendNATSRequestInterceptor(client, func(string) (string, bool) { return "fluffyhdr.echo", true })

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer abc", "x-request-id", "req-1")
	reply := &wrapperspb.StringValue{}
	require.NoError(t, invoke(ctx, "/svc/Echo", &wrapperspb.StringValue{}, reply, nil, nil))
	require.Equal(t, "Bearer abc|req-1", reply.GetValue())

	// and nothing is invented when the caller sent none
	reply = &wrapperspb.StringValue{}
	require.NoError(t, invoke(context.Background(), "/svc/Echo", &wrapperspb.StringValue{}, reply, nil, nil))
	require.Equal(t, "|", reply.GetValue())
}
