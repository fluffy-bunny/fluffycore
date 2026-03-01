package grpcclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	log "github.com/rs/zerolog/log"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	oauth2 "golang.org/x/oauth2"
	grpc "google.golang.org/grpc"
	backoff "google.golang.org/grpc/backoff"
	credentials "google.golang.org/grpc/credentials"
	insecure "google.golang.org/grpc/credentials/insecure"
	oauth "google.golang.org/grpc/credentials/oauth"
	datadog_grpctrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/google.golang.org/grpc"
)

// Example Usage:
// ```
/*
func Test() error {
	// Create the GRPC Client
	grpcCli, err := grpcclient.NewGrpcClient(
		grpcclient.WithHost("1.2.3.4"),
		grpcclient.WithPort(50051),

		// Option 1. Using user token
		grpcclient.WithToken(token.GetToken()), // Get token from runtime: token, err := runtime.GetSingleton().GetAuthTokenAct(ctx)

		// Option 2. Using microservice credentials
		grpcclient.WithTokenSource(tokenSource), // Provider must be initialized on microservice startup and shared among all clients.
	)

	if err != nil {
		log.Error().Err(err).Msg("Creating gRPC client")
		return err
	}

	defer grpcCli.Close()

	// Create the gRPC client
	cli := api.NewAppServiceClient(grpcCli.GetConnection())

	// Make the function call
	ctx, cancel := grpcclient.ContextWithTimeout(ctx)
	defer cancel()
	reqOutput, err := cli.GetAvailable(ctx, input)
	if err != nil {
		log.Fatal().Err(err).Msg("GetAvailable call failed")
	}
}
*/
// ```

var defaultGrpcCallTimeoutInSeconds *int

// GrpcClient object
type GrpcClient struct {
	conn                 *grpc.ClientConn
	target               string
	authority            string
	host                 string
	port                 int
	insecure             bool
	sidecarSecured       bool
	certBundleFile       string
	clientCerts          []tls.Certificate
	ctx                  context.Context
	tokenSource          oauth2.TokenSource
	enableOTELTracing    bool
	enableDataDogTracing bool
}

// ClientOption is used for option pattern calling
type GrpcClientOption func(*GrpcClient) error

// Create a client to access other microservices that expose grpc
// Do not use this client to call external systems since it create insecure channel. Envoy provides security for internal networking.
func NewGrpcClient(opts ...GrpcClientOption) (*GrpcClient, error) {
	// Create a client
	c := &GrpcClient{
		insecure:          true, // By default Envoy cares about security
		sidecarSecured:    true, // TODO: sidecarSecured/insecure should be set based on a cmdline/env option
		enableOTELTracing: true,
	}

	// Process options
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			log.Error().Err(err).Msg("ClientOption error")
			return nil, err
		}
	}

	dialOpts := []grpc.DialOption{
		grpc.WithDisableRetry(),
		grpc.WithConnectParams(grpc.ConnectParams{MinConnectTimeout: time.Second, Backoff: backoff.DefaultConfig}),
	}
	if !fluffycore_utils.IsEmptyOrNil(c.authority) {
		dialOpts = append(dialOpts, grpc.WithAuthority(c.authority))
	}
	tracingOpsFuncs := map[bool]func(){
		c.enableOTELTracing: func() {
			dialOpts = append(dialOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
		},
		c.enableDataDogTracing: func() {
			streamTraceInterceptor := datadog_grpctrace.StreamClientInterceptor()
			unaryTraceInterceptor := datadog_grpctrace.UnaryClientInterceptor()

			dialOpts = append(dialOpts,
				grpc.WithStreamInterceptor(streamTraceInterceptor),
				grpc.WithUnaryInterceptor(unaryTraceInterceptor))
		},
	}
	for k, v := range tracingOpsFuncs {
		if k {
			v()
			break
		}
	}

	// Let user choose whether to put full address or to put host & port separately
	url := c.target
	if url == "" {
		url = fmt.Sprintf("%s:%d", c.host, c.port)
	}

	// Do we need to use a custom server root cert bundle?
	if !c.insecure {
		var serverNameOverride string
		hostParts := strings.Split(url, ":")
		if len(hostParts) == 2 && hostParts[0] != "" && !strings.Contains(hostParts[0], ".") {
			serverNameOverride = hostParts[0]
		}

		if len(c.certBundleFile) > 0 {
			trnCreds, err := credentials.NewClientTLSFromFile(c.certBundleFile, serverNameOverride)
			if err != nil {
				log.Error().Err(err).Str("url", url).Msg("Failed to connect")
				return nil, err
			}

			dialOpts = append(dialOpts, grpc.WithTransportCredentials(trnCreds))
		} else {
			certPool := x509.NewCertPool()
			tlsConfig := &tls.Config{
				ServerName: serverNameOverride,
				RootCAs:    certPool,
			}

			// Do we have client certs for mTLS?
			if len(c.clientCerts) > 0 {
				tlsConfig.Certificates = c.clientCerts
			}

			trnCreds := credentials.NewTLS(tlsConfig)

			dialOpts = append(dialOpts, grpc.WithTransportCredentials(trnCreds))
		}

		// Do we have auth to pass?
		if fluffycore_utils.IsNotNil(c.tokenSource) {
			rpcCreds := oauth.TokenSource{TokenSource: c.tokenSource}
			dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(rpcCreds))
		}

	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

		if fluffycore_utils.IsNotNil(c.tokenSource) {
			// this wins
			rpcCreds := NewOauthAccessFromTokenSource(c.tokenSource, c.sidecarSecured)
			dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(rpcCreds))
		}
	}

	// Set up a connection to the server.
	var err error
	c.conn, err = grpc.NewClient(url, dialOpts...)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to connect")
		return nil, err
	}

	return c, nil
}

// Close tears down the Client and all underlying connections.
func (c *GrpcClient) Close() error {
	return c.conn.Close()
}

// GetConnection returns the underlying grpc connection
func (c *GrpcClient) GetConnection() *grpc.ClientConn {
	return c.conn
}

//
// Options
//

// WithDataDogTracer returns a GrpcClientOption that enables or disables Datadog tracing.
func WithDataDogTracer(enable bool) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.enableDataDogTracing = enable
		return nil
	}
}

// Deprecated: Use WithDataDogTracer instead.
func WithDataDpgTracer(enable bool) GrpcClientOption {
	return WithDataDogTracer(enable)
}

// WithOTELTracer returns a GrpcClientOption that enables or disables OpenTelemetry tracing.
func WithOTELTracer(enable bool) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.enableOTELTracing = enable
		return nil
	}
}

// Sets full url to gRPC endpoint. Do not this method with WithHost or WithPort.
func WithTarget(target string) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.target = target
		return nil
	}
}

// WithAuthority returns a GrpcClientOption that sets the :authority header value for gRPC calls.
func WithAuthority(authority string) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.authority = authority
		return nil
	}
}

// Sets host. Use with WithPort
func WithHost(host string) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.host = host
		return nil
	}
}

// Sets port. Use with WithHost method
func WithPort(port int) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.port = port
		return nil
	}
}

// WithTokenSource sets the token source to use for authentication.
func WithTokenSource(tokenSource oauth2.TokenSource) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.tokenSource = tokenSource
		return nil
	}
}

// WithCertBundle returns a GrpcClientOption that sets the CA certificate bundle file for TLS.
func WithCertBundle(certBundleFile string) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.certBundleFile = certBundleFile
		return nil
	}
}

// WithClientCert returns a GrpcClientOption that appends a client TLS certificate.
func WithClientCert(clientCert tls.Certificate) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.clientCerts = append(c.clientCerts, clientCert)
		return nil
	}
}

// WithInsecure returns a GrpcClientOption that enables or disables insecure (non-TLS) connections.
func WithInsecure(insecure bool) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.insecure = insecure
		return nil
	}
}

// WithSidecarSecured returns a GrpcClientOption for connections secured by a sidecar proxy (e.g., Envoy).
func WithSidecarSecured(sidecarSecured bool) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.insecure = sidecarSecured
		c.sidecarSecured = sidecarSecured
		return nil
	}
}

// WithContext will use DialContext instead of Dial
func WithContext(ctx context.Context) GrpcClientOption {
	return func(c *GrpcClient) error {
		c.ctx = ctx
		return nil
	}
}

//
// Global settings
//

// SetDefaultGrpcCallTimeout sets the global default timeout in seconds for gRPC calls.
func SetDefaultGrpcCallTimeout(timeoutInSeconds int) {
	defaultGrpcCallTimeoutInSeconds = &timeoutInSeconds
}

//
// Helpers
//

// Creates context with timeout.
func ContextWithTimeout(ctx context.Context, duration ...time.Duration) (context.Context, context.CancelFunc) {
	var timeoutDuration time.Duration
	if len(duration) > 0 {
		timeoutDuration = duration[0]
	} else if defaultGrpcCallTimeoutInSeconds != nil {
		timeoutDuration = time.Second * time.Duration(*defaultGrpcCallTimeoutInSeconds)
	} else {
		panic("GRPC client: Default grpc call timeout was not set")
	}

	return context.WithTimeout(ctx, timeoutDuration)
}
