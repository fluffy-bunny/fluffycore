package nats_micro_service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	codes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// Headers carrying the gRPC status of a failed NATS micro request, next to NATS micro's own
// Nats-Service-Error / Nats-Service-Error-Code. Error replies used to be "500" (or "400")
// with the message only, and clients turned every error into codes.Internal, so NotFound,
// PermissionDenied, InvalidArgument and friends were all lost in transit.
const (
	GRPCStatusHeader  = "Grpc-Status"
	GRPCMessageHeader = "Grpc-Message"
)

// grpcToServiceCode maps a gRPC code to the HTTP-style code NATS micro reports in
// Nats-Service-Error-Code, for clients that don't read GRPCStatusHeader.
func grpcToServiceCode(c codes.Code) string {
	switch c {
	case codes.InvalidArgument, codes.OutOfRange:
		return "400"
	case codes.Unauthenticated:
		return "401"
	case codes.PermissionDenied:
		return "403"
	case codes.NotFound:
		return "404"
	case codes.AlreadyExists, codes.Aborted:
		return "409"
	case codes.FailedPrecondition:
		return "412"
	case codes.ResourceExhausted:
		return "429"
	case codes.Canceled:
		return "499"
	case codes.Unimplemented:
		return "501"
	case codes.Unavailable:
		return "503"
	case codes.DeadlineExceeded:
		return "504"
	}
	return "500"
}

// serviceCodeToGRPC maps an HTTP-style Nats-Service-Error-Code (from a service that doesn't
// send GRPCStatusHeader) back to a gRPC code.
func serviceCodeToGRPC(code string) codes.Code {
	switch code {
	case "400":
		return codes.InvalidArgument
	case "401":
		return codes.Unauthenticated
	case "403":
		return codes.PermissionDenied
	case "404":
		return codes.NotFound
	case "409":
		return codes.AlreadyExists
	case "412":
		return codes.FailedPrecondition
	case "429":
		return codes.ResourceExhausted
	case "499":
		return codes.Canceled
	case "501":
		return codes.Unimplemented
	case "503":
		return codes.Unavailable
	case "504", "408":
		return codes.DeadlineExceeded
	}
	return codes.Internal
}

// headerSafe makes s usable as a header value (no line breaks).
func headerSafe(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}

// respondError replies to req with err's gRPC status. It always responds: NATS micro's
// Error refuses an empty description without sending anything, which left the caller
// waiting for its full timeout.
func respondError(req micro.Request, err error) {
	st := grpcstatus.Convert(err)
	msg := st.Message()
	if msg == "" {
		msg = st.Code().String()
	}
	msg = headerSafe(msg)
	_ = req.Error(grpcToServiceCode(st.Code()), msg, nil, micro.WithHeaders(micro.Headers{
		GRPCStatusHeader:  []string{strconv.Itoa(int(st.Code()))},
		GRPCMessageHeader: []string{msg},
	}))
}

// statusFromNATSResponse returns the error a NATS micro reply carries, or nil for a success.
func statusFromNATSResponse(msg *nats.Msg) error {
	if msg == nil || msg.Header == nil {
		return nil
	}
	description := strings.Join(msg.Header.Values(micro.ErrorHeader), "; ")
	serviceCode := msg.Header.Get(micro.ErrorCodeHeader)
	grpcCode := msg.Header.Get(GRPCStatusHeader)
	if description == "" && serviceCode == "" && grpcCode == "" {
		return nil
	}
	if message := msg.Header.Get(GRPCMessageHeader); message != "" {
		description = message
	}
	if n, err := strconv.Atoi(grpcCode); err == nil && n > 0 && n <= int(codes.Unauthenticated) {
		return grpcstatus.Error(codes.Code(n), description)
	}
	return grpcstatus.Error(serviceCodeToGRPC(serviceCode), description)
}

// validateSubjectToken rejects values that would change a subject's structure when
// substituted as one token: empty, or containing '.', the wildcards '*' and '>', whitespace
// or control characters.
func validateSubjectToken(name, value string) error {
	if value == "" {
		return grpcstatus.Errorf(codes.InvalidArgument, "subject token %q is empty", name)
	}
	for _, r := range value {
		if r == '.' || r == '*' || r == '>' || r <= ' ' || r == 0x7f {
			return grpcstatus.Errorf(codes.InvalidArgument, "subject token %q contains an invalid character %q", name, r)
		}
	}
	return nil
}

// statusFromTransportError gives a failed NATS request a gRPC status, so callers can tell
// "nobody is listening" or "ran out of time" from a real failure. It used to be a plain
// wrapped error, which gRPC reports as Unknown.
func statusFromTransportError(err error) error {
	switch {
	case errors.Is(err, nats.ErrNoResponders):
		return grpcstatus.Errorf(codes.Unavailable, "nats request failed: %v", err)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, nats.ErrTimeout):
		return grpcstatus.Errorf(codes.DeadlineExceeded, "nats request failed: %v", err)
	case errors.Is(err, context.Canceled):
		return grpcstatus.Errorf(codes.Canceled, "nats request failed: %v", err)
	case errors.Is(err, nats.ErrConnectionClosed), errors.Is(err, nats.ErrDisconnected):
		return grpcstatus.Errorf(codes.Unavailable, "nats request failed: %v", err)
	}
	return grpcstatus.Errorf(codes.Unknown, "nats request failed: %v", err)
}
