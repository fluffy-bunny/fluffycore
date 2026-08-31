package config

type (
	CoreConfig struct {
		ApplicationEnvironment string `json:"applicationEnvironment" mapstructure:"APPLICATION_ENVIRONMENT"`
		ApplicationName        string `json:"applicationName" mapstructure:"APPLICATION_NAME"`
		PORT                   int    `json:"port" mapstructure:"PORT"`
		GRPCGateWayEnabled     bool   `json:"grpcGateWayEnabled" mapstructure:"GRPC_GATEWAY_ENABLED"`
		RESTPort               int    `json:"restPort" mapstructure:"REST_PORT"`
		PrettyLog              bool   `json:"prettyLog" mapstructure:"PRETTY_LOG"`
		LogLevel               string `json:"logLevel" mapstructure:"LOG_LEVEL"`
		NATSEnabled            bool   `json:"enableNats" mapstructure:"NATS_ENABLED"`

		// TLSEnabled turns on TLS (and, when TLSClientCAFile is set, mutual TLS)
		// for the gRPC server. See runtime/servertls for how these are consumed.
		TLSEnabled bool `json:"tlsEnabled" mapstructure:"tlsEnabled"`
		// TLSCertFile / TLSKeyFile are PEM file paths for the server's own
		// certificate and private key. Required when TLSEnabled is true. Both
		// files are re-read from disk on the fly when their contents change, so
		// an external agent (HashiCorp Vault Agent, the Vault Secrets Operator/CSI
		// provider, cert-manager, ...) rotating them on disk takes effect without
		// restarting the process.
		TLSCertFile string `json:"tlsCertFile" mapstructure:"tlsCertFile"`
		TLSKeyFile  string `json:"tlsKeyFile" mapstructure:"tlsKeyFile"`
		// TLSClientCAFile is a PEM file of CA certificate(s) the server trusts to
		// have issued a client certificate. Setting this is what turns on mutual
		// TLS: the server will request a client certificate and verify it against
		// this bundle. Leave empty for plain (server-only) TLS. Reloaded on
		// change, same as the cert/key above.
		TLSClientCAFile string `json:"tlsClientCAFile" mapstructure:"tlsClientCAFile"`
		// TLSClientAuth controls how strictly a client certificate is required:
		// "none" (default when TLSClientCAFile is empty) -- never request one;
		// "request" -- ask for one but never verify or require it (rarely useful);
		// "verify_if_given" (default when TLSClientCAFile is set) -- request one,
		// verify it against TLSClientCAFile if the client offers it, but don't
		// fail the handshake if it doesn't -- this is what enables *method-level*
		// mTLS enforcement: see middleware/auth/mtls's README for gating
		// individual methods on the resulting claims via your existing
		// per-method IEntryPointConfig/claims-AST authorization;
		// "require" -- reject the TLS handshake outright unless the client
		// presents a certificate that verifies, for every method, connection-wide.
		TLSClientAuth string `json:"tlsClientAuth" mapstructure:"tlsClientAuth"`
		// TLSCertsDir is a convenience shortcut for the three file paths above:
		// when set, it implies TLSEnabled and fills in whichever of
		// TLSCertFile/TLSKeyFile/TLSClientCAFile were left empty with
		// <TLSCertsDir>/server-cert.pem, <TLSCertsDir>/server-key.pem, and
		// <TLSCertsDir>/ca.pem respectively -- the exact layout `gencert`
		// writes. One config value turns mTLS on, pointed at a directory,
		// instead of four. Any of the three explicit fields you do set still
		// wins over the derived path, so this is safe to leave on a shared
		// default (e.g. "./certs") while a real deployment overrides the
		// individual fields with Vault-injected paths.
		TLSCertsDir string `json:"tlsCertsDir" mapstructure:"tlsCertsDir"`
	}
)
