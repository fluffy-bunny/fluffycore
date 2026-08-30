package certgen

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// Forever is a validity period long enough to be treated as "never expires"
// for a checked-in development CA/certificate set (100 years). X.509 has no
// literal "no expiry" -- RFC 5280 requires a NotAfter -- so this is the
// conventional stand-in for cert bundles that must not need periodic
// regeneration, e.g. this repo's checked-in ./certs. Never use this for
// anything production-facing; a real CA (HashiCorp Vault's PKI secrets
// engine, cert-manager, ...) should issue short-lived certificates instead.
const Forever = 100 * 365 * 24 * time.Hour

// BundleOptions configures a CA + server certificate + client certificate set,
// as needed to exercise mutual TLS end-to-end (see Bundle).
type BundleOptions struct {
	ServerCommonName string
	// ServerHosts are SANs for the server certificate; each is parsed as an IP
	// literal if possible, otherwise treated as a DNS name.
	ServerHosts      []string
	ClientCommonName string
	// ClientURIs are raw URI SANs for the client certificate, e.g. a SPIFFE ID
	// such as "spiffe://cluster.local/ns/foo/sa/bar".
	ClientURIs []string
	Org        string
	// ValidFor is the leaf certificates' validity period; the CA is valid 5x
	// as long. Defaults to 90 days when <= 0.
	ValidFor time.Duration
}

// Bundle is a self-signed CA plus a server certificate and a client
// certificate it issued -- everything needed for a real client-to-server
// mutual TLS handshake in a test.
type Bundle struct {
	CA     *CA
	Server *Cert
	Client *Cert
}

// bundleFileNames is the fixed, shared file layout WriteFiles uses -- both
// the cobracore and cmd/cli `gencert` commands rely on this exact set.
var bundleFileNames = []string{"ca.pem", "ca-key.pem", "server-cert.pem", "server-key.pem", "client-cert.pem", "client-key.pem"}

// NewBundle generates a fresh CA, server certificate, and client certificate
// per opts.
func NewBundle(opts BundleOptions) (*Bundle, error) {
	validFor := opts.ValidFor
	if validFor <= 0 {
		validFor = 90 * 24 * time.Hour
	}

	ca, err := NewCA(CAOptions{
		CommonName: "fluffycore-dev-ca",
		Org:        opts.Org,
		ValidFor:   caValidForFrom(validFor),
	})
	if err != nil {
		return nil, fmt.Errorf("certgen: generate CA: %w", err)
	}

	dnsNames, ips := splitHosts(opts.ServerHosts)
	server, err := NewLeafCert(ca, CertOptions{
		CommonName:  opts.ServerCommonName,
		Org:         opts.Org,
		DNSNames:    dnsNames,
		IPAddresses: ips,
		ValidFor:    validFor,
		IsServer:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("certgen: generate server certificate: %w", err)
	}

	parsedClientURIs, err := parseURIs(opts.ClientURIs)
	if err != nil {
		return nil, err
	}
	client, err := NewLeafCert(ca, CertOptions{
		CommonName: opts.ClientCommonName,
		Org:        opts.Org,
		URIs:       parsedClientURIs,
		ValidFor:   validFor,
		IsServer:   false,
	})
	if err != nil {
		return nil, fmt.Errorf("certgen: generate client certificate: %w", err)
	}

	return &Bundle{CA: ca, Server: server, Client: client}, nil
}

// WriteFiles writes the bundle's 6 PEM files (ca.pem, ca-key.pem,
// server-cert.pem, server-key.pem, client-cert.pem, client-key.pem) into dir,
// creating it if needed. If force is false and any of those files already
// exist, it returns an error instead of overwriting anything.
func (b *Bundle) WriteFiles(dir string, force bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("certgen: create output directory %s: %w", dir, err)
	}
	if !force {
		for _, f := range bundleFileNames {
			if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
				return fmt.Errorf("certgen: %s already exists in %s -- pass force=true to overwrite", f, dir)
			}
		}
	}

	writes := map[string][]byte{
		"ca.pem":          b.CA.CertPEM,
		"ca-key.pem":      b.CA.KeyPEM,
		"server-cert.pem": b.Server.CertPEM,
		"server-key.pem":  b.Server.KeyPEM,
		"client-cert.pem": b.Client.CertPEM,
		"client-key.pem":  b.Client.KeyPEM,
	}
	for _, name := range bundleFileNames {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, writes[name], 0o600); err != nil {
			return fmt.Errorf("certgen: write %s: %w", path, err)
		}
	}
	return nil
}

// caValidForFrom returns 5x leafValidFor -- a CA conventionally outlives what
// it signs by a comfortable margin -- unless that multiplication would
// overflow time.Duration's int64 nanosecond range (anything above roughly
// 58 years), which it does for e.g. Forever (100 years). Detected via the
// standard "multiply then divide back" check; on overflow the CA simply
// matches the leaf's validity instead of exceeding it -- still never shorter
// than what it signs, which is the actual requirement, just not padded 5x.
func caValidForFrom(leafValidFor time.Duration) time.Duration {
	const multiplier = 5
	caValidFor := leafValidFor * multiplier
	if caValidFor/multiplier != leafValidFor {
		return leafValidFor
	}
	return caValidFor
}

func splitHosts(hosts []string) (dnsNames []string, ips []net.IP) {
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			ips = append(ips, ip)
		} else {
			dnsNames = append(dnsNames, h)
		}
	}
	return dnsNames, ips
}

func parseURIs(raw []string) ([]*url.URL, error) {
	var parsed []*url.URL
	for _, u := range raw {
		p, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("certgen: invalid URI %q: %w", u, err)
		}
		parsed = append(parsed, p)
	}
	return parsed, nil
}
