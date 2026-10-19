package nats_micro_service

import (
	"context"
	"errors"
	"testing"

	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	"github.com/stretchr/testify/require"
	codes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	proto "google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// fakeRequest records how a handler replied, the way micro's own request would build it.
type fakeRequest struct {
	subject string
	data    []byte
	replied *nats.Msg
	replies int
}

func (f *fakeRequest) Respond(b []byte, opts ...micro.RespondOpt) error {
	m := &nats.Msg{Data: b}
	for _, o := range opts {
		o(m)
	}
	f.replied, f.replies = m, f.replies+1
	return nil
}
func (f *fakeRequest) RespondJSON(any, ...micro.RespondOpt) error { return errors.New("unused") }
func (f *fakeRequest) Error(code, description string, data []byte, opts ...micro.RespondOpt) error {
	if code == "" || description == "" { // micro refuses these without responding
		return errors.New("argument required")
	}
	m := &nats.Msg{Header: nats.Header{micro.ErrorHeader: {description}, micro.ErrorCodeHeader: {code}}, Data: data}
	for _, o := range opts {
		o(m)
	}
	f.replied, f.replies = m, f.replies+1
	return nil
}
func (f *fakeRequest) Data() []byte           { return f.data }
func (f *fakeRequest) Headers() micro.Headers { return micro.Headers{} }
func (f *fakeRequest) Subject() string        { return f.subject }
func (f *fakeRequest) Reply() string          { return "" }

// Every service error used to come back as codes.Internal.
func TestStatusRoundTrip(t *testing.T) {
	for _, c := range []codes.Code{codes.NotFound, codes.PermissionDenied, codes.InvalidArgument, codes.Unauthenticated, codes.Unavailable} {
		req := &fakeRequest{}
		respondError(req, grpcstatus.Error(c, "why"))
		require.Equal(t, 1, req.replies)
		got := statusFromNATSResponse(req.replied)
		require.Equal(t, c, grpcstatus.Code(got), "code %v", c)
		require.Equal(t, "why", grpcstatus.Convert(got).Message())
		require.Equal(t, grpcToServiceCode(c), req.replied.Header.Get(micro.ErrorCodeHeader))
	}

	// A plain error with an empty message still gets a reply (micro would drop it).
	req := &fakeRequest{}
	respondError(req, errors.New(""))
	require.Equal(t, 1, req.replies)
	require.Error(t, statusFromNATSResponse(req.replied))

	require.NoError(t, statusFromNATSResponse(&nats.Msg{Data: []byte("x")}), "no error headers = success")
}

// A non-fluffycore NATS micro service sends only Nats-Service-Error(-Code).
func TestStatusFromForeignService(t *testing.T) {
	msg := &nats.Msg{Header: nats.Header{micro.ErrorHeader: {"nope"}, micro.ErrorCodeHeader: {"403"}}}
	err := statusFromNATSResponse(msg)
	require.Equal(t, codes.PermissionDenied, grpcstatus.Code(err))
	require.Equal(t, "nope", grpcstatus.Convert(err).Message())
	msg = &nats.Msg{Header: nats.Header{micro.ErrorHeader: {"boom"}, micro.ErrorCodeHeader: {"500"}}}
	require.Equal(t, codes.Internal, grpcstatus.Code(statusFromNATSResponse(msg)))
}

type noRoute struct{}

func handle(req micro.Request, protoMsg func() (protoreflect.ProtoMessage, error), method func(context.Context, *wrapperspb.StringValue) (*wrapperspb.StringValue, error)) {
	HandleRequest(req, "", &NATSMicroHandlerInfo{ParameterizedToken: "svc"},
		func(r *wrapperspb.StringValue) error { return nil },
		protoMsg,
		func(m protoreflect.ProtoMessage, r *wrapperspb.StringValue) error { return nil },
		method)
}

func okProto() (protoreflect.ProtoMessage, error) { return &wrapperspb.StringValue{}, nil }

// HandleRequest used to return without replying when protoRequestMessage failed, and turned
// every service error into "500".
func TestHandleRequest_AlwaysRespondsWithStatus(t *testing.T) {
	req := &fakeRequest{subject: "svc"}
	handle(req, func() (protoreflect.ProtoMessage, error) { return nil, errors.New("bad descriptor") }, nil)
	require.Equal(t, 1, req.replies, "the caller must not be left to time out")
	require.Equal(t, codes.Internal, grpcstatus.Code(statusFromNATSResponse(req.replied)))

	req = &fakeRequest{subject: "svc"}
	handle(req, okProto, func(context.Context, *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
		return nil, grpcstatus.Error(codes.NotFound, "no such org")
	})
	require.Equal(t, codes.NotFound, grpcstatus.Code(statusFromNATSResponse(req.replied)))
	require.Equal(t, "404", req.replied.Header.Get(micro.ErrorCodeHeader))

	req = &fakeRequest{subject: "svc"}
	handle(req, okProto, func(context.Context, *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
		return &wrapperspb.StringValue{}, nil // all defaults: marshals to zero bytes
	})
	require.NoError(t, statusFromNATSResponse(req.replied))
	require.Empty(t, req.replied.Data)
}

// Request fields go into subjects; they must not be able to change the subject's shape.
func TestReplaceTokens_RejectsSubjectInjection(t *testing.T) {
	for _, v := range []string{"a.b", "*", ">", "a b", ""} {
		_, err := ReplaceTokens("org.${name}", &descriptorpb.FileDescriptorProto{Name: proto.String(v)})
		require.Equal(t, codes.InvalidArgument, grpcstatus.Code(err), "value %q", v)
	}
	s, err := ReplaceTokens("org.${name}", &descriptorpb.FileDescriptorProto{Name: proto.String("acme-1")})
	require.NoError(t, err)
	require.Equal(t, "org.acme-1", s)
}
