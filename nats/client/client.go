package client

import (
	"context"
	"time"

	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	nats "github.com/nats-io/nats.go"
	oauth2 "golang.org/x/oauth2"
	metadata "google.golang.org/grpc/metadata"
)

type (
	NATSClient struct {
		conn         *nats.Conn
		tokenSource  oauth2.TokenSource
		timeout      time.Duration
		ctxModifiers []ContextModifier
		callOptions  []CallOption
	}
	CallInfo struct {
		Ctx     context.Context
		Subject string
	}
	CallOption interface {
		before(*CallInfo) error
		after(*CallInfo) error
	}
)
type ContextModifier func(ctx context.Context, subject string) (context.Context, error)

func EnsureOutboundSpanTracing(ctx context.Context, subject string) (context.Context, error) {
	// try to pull the correlation id .  wellknown.XCorrelationIDName
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.Pairs()
	}
	xData := md.Get(wellknown.XCorrelationIDName)
	if fluffycore_utils.IsEmptyOrNil(xData) {
		correlationID := fluffycore_utils.GenerateUniqueID()
		md.Set(wellknown.XCorrelationIDName, correlationID)
	}
	// turn x-span into x-parent
	xData = md.Get(wellknown.XSpanName)
	if fluffycore_utils.IsNotEmptyOrNil(xData) {
		md.Delete(wellknown.XParentName)
		md.Set(wellknown.XParentName, xData...)
		md.Delete(wellknown.XSpanName)
	}
	xSpan := fluffycore_utils.GenerateUniqueID()
	md.Set(wellknown.XSpanName, xSpan)

	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx, nil
}

// NATSClientOption is used for option pattern calling
type NATSClientOption func(*NATSClient) error

func NewNATSClient(opts ...NATSClientOption) (*NATSClient, error) {
	client := &NATSClient{}
	client.timeout = 5 * time.Second // default
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}
	return client, nil
}

// WithNATSClientConn can be a custom solution with nats auth callouts.
// This is the nats connection to the nats server.
func WithNATSClientConn(conn *nats.Conn) NATSClientOption {
	return func(c *NATSClient) error {
		c.conn = conn
		return nil
	}
}

func WithCallOptions(callOptions []CallOption) NATSClientOption {
	return func(c *NATSClient) error {
		c.callOptions = callOptions
		return nil
	}
}

// WithContextModifiers is a list of context interceptors
func WithContextModifiers(ctxModifiers ...ContextModifier) NATSClientOption {
	return func(c *NATSClient) error {
		c.ctxModifiers = ctxModifiers
		return nil
	}
}

// WithTimeout sets the timeout for the nats client
func WithTimeout(timeout time.Duration) NATSClientOption {
	return func(c *NATSClient) error {
		c.timeout = timeout
		return nil
	}
}

// WithTokenSource is the token needed to talk to the nats micro service handlers.
func WithTokenSource(tokenSource oauth2.TokenSource) NATSClientOption {
	return func(c *NATSClient) error {
		c.tokenSource = tokenSource
		return nil
	}
}

func (s *NATSClient) createNATSRequestHeaders(ctx context.Context) (nats.Header, error) {

	// we are a client so the metadata here is outgoing.  we just need to propogate that to
	// the nats headers
	var err error
	for _, ctxModifier := range s.ctxModifiers {
		ctx, err = ctxModifier(ctx, "")
		if err != nil {
			return nil, err
		}
	}
	// Read metadata AFTER modifiers have had a chance to update the context
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.Pairs()
	}
	headers := nats.Header{}
	// propogate the grpc metadata to the nats headers

	for k, v := range md {
		headers[k] = v
	}

	if s.tokenSource != nil {
		token, err := s.tokenSource.Token()
		if err != nil {
			return nil, err
		}
		headers.Set("Authorization", "Bearer "+token.AccessToken)
	}
	return headers, nil
}

// Close tears down the Client and all underlying connections.
func (s *NATSClient) Close() error {
	s.conn.Close()
	return nil
}

// GetConnection returns the underlying nats connection
func (s *NATSClient) GetConnection() *nats.Conn {
	return s.conn
}

// RequestWithContext sends a request and waits for a response
func (s *NATSClient) RequestWithContext(ctx context.Context,
	subject string, msg []byte) (*nats.Msg, error) {
	headers, err := s.createNATSRequestHeaders(ctx)
	if err != nil {
		return nil, err
	}

	// Prepare NATS message
	natsMessage := &nats.Msg{
		Subject: subject,
		Data:    msg,
		Header:  headers,
	}
	CallInfo := &CallInfo{
		Ctx:     ctx,
		Subject: subject,
	}
	for _, opt := range s.callOptions {
		if err := opt.before(CallInfo); err != nil {
			return nil, err
		}
	}
	// Send request and wait for response
	natsResponse, err := s.conn.RequestMsg(natsMessage, s.timeout)
	if err != nil {
		return nil, err
	}

	for _, opt := range s.callOptions {
		if err := opt.after(CallInfo); err != nil {
			return nil, err
		}
	}
	return natsResponse, nil
}
