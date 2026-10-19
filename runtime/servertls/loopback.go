package servertls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	credentials "google.golang.org/grpc/credentials"
	insecure "google.golang.org/grpc/credentials/insecure"
)

// LoopbackServerName is the name loopback clients verify the server certificate against.
// The server certificate must cover it (gencert's does: DNS:localhost, IP:127.0.0.1).
const LoopbackServerName = "localhost"

// LoopbackTarget returns the gRPC dial target for reaching a server listening on addr from
// the same process: an unspecified bind address (0.0.0.0, ::) becomes localhost, since the
// wildcard address is not something a client can connect to everywhere (Windows refuses
// 0.0.0.0). Unix sockets use the unix: scheme. For other listener types (e.g. an in-memory
// listener) it returns localhost:fallbackPort.
func LoopbackTarget(addr net.Addr, fallbackPort int) string {
	switch a := addr.(type) {
	case *net.TCPAddr:
		host := a.IP.String()
		if a.IP == nil || a.IP.IsUnspecified() {
			host = LoopbackServerName
		}
		return net.JoinHostPort(host, strconv.Itoa(a.Port))
	case *net.UnixAddr:
		return "unix:" + a.Name
	}
	return net.JoinHostPort(LoopbackServerName, strconv.Itoa(fallbackPort))
}

// LoopbackClientCredentials returns transport credentials for an in-process client (the
// grpc-gateway, healthz, the NATS bridge) dialing this server. serverTLS says whether the
// server is actually serving TLS (see FluffyCoreOTELStartup.ServerTLSEnabled): plaintext when
// it isn't.
//
// With TLS, the server's own certificate file is the trust anchor, re-read at every
// handshake so a hot-reloaded certificate keeps working, and the certificate is verified for
// LoopbackServerName. When the server requires client certificates (tlsClientAuth=require)
// a loopback client can't satisfy that; RequiresClientCert reports it so callers can warn.
func LoopbackClientCredentials(cfg *fluffycore_contracts_config.CoreConfig, serverTLS bool) (credentials.TransportCredentials, error) {
	if !serverTLS {
		return insecure.NewCredentials(), nil
	}
	certFile, ok := LoopbackTrustFile(cfg)
	if !ok {
		return nil, errors.New("servertls: server TLS is on but no server certificate file is configured")
	}
	return credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: LoopbackServerName,
		// Chain and name verification happen in VerifyConnection instead, against a trust
		// pool rebuilt from certFile at each handshake. Not "insecure": nothing is accepted
		// that doesn't verify.
		InsecureSkipVerify: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			return verifyAgainstFile(cs, certFile)
		},
	}), nil
}

func verifyAgainstFile(cs tls.ConnectionState, certFile string) error {
	if len(cs.PeerCertificates) == 0 {
		return errors.New("servertls: loopback peer presented no certificate")
	}
	pem, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("servertls: reading loopback trust file: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return fmt.Errorf("servertls: no certificates in %s", certFile)
	}
	intermediates := x509.NewCertPool()
	for _, c := range cs.PeerCertificates[1:] {
		intermediates.AddCert(c)
	}
	_, err = cs.PeerCertificates[0].Verify(x509.VerifyOptions{
		DNSName:       LoopbackServerName,
		Roots:         roots,
		Intermediates: intermediates,
	})
	return err
}

// RequiresClientCert reports whether cfg makes the server reject clients without a
// verified certificate (tlsClientAuth=require) -- which in-process loopback clients can't
// present.
func RequiresClientCert(cfg *fluffycore_contracts_config.CoreConfig) bool {
	if cfg == nil {
		return false
	}
	r := resolveConfig(cfg)
	if !r.enabled {
		return false
	}
	auth, err := parseClientAuth(r.clientAuth, r.clientCAFile)
	return err == nil && auth == tls.RequireAndVerifyClientCert
}
