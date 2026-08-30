// Package certgen creates self-signed CA certificates and leaf certificates
// issued by them, entirely in-process (ECDSA P-256, no external tooling). It
// backs the `gencert` CLI command (cobracore/cmd/gencert) used to produce a
// local CA/server/client certificate set for testing mutual TLS, and the tests
// for runtime/servertls and middleware/auth/mtls.
//
// It is not meant for issuing production certificates -- for that, use a real
// CA (HashiCorp Vault's PKI secrets engine, cert-manager, your org's internal
// CA, ...).
package certgen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"time"
)

type (
	// CAOptions configures a self-signed CA certificate.
	CAOptions struct {
		CommonName string
		Org        string
		// ValidFor defaults to 5 years when <= 0.
		ValidFor time.Duration
	}

	// CertOptions configures a leaf certificate issued by a CA.
	CertOptions struct {
		CommonName  string
		Org         string
		DNSNames    []string
		IPAddresses []net.IP
		// URIs are URI SANs, e.g. a SPIFFE ID ("spiffe://cluster.local/ns/foo/sa/bar").
		URIs []*url.URL
		// ValidFor defaults to 90 days when <= 0.
		ValidFor time.Duration
		// IsServer sets ExtKeyUsageServerAuth; otherwise ExtKeyUsageClientAuth.
		IsServer bool
	}

	// CA holds a self-signed CA's certificate and ECDSA P-256 key, parsed and
	// PEM-encoded.
	CA struct {
		Cert    *x509.Certificate
		Key     *ecdsa.PrivateKey
		CertPEM []byte
		KeyPEM  []byte
	}

	// Cert holds a leaf certificate and ECDSA P-256 key issued by a CA, parsed
	// and PEM-encoded.
	Cert struct {
		Cert    *x509.Certificate
		Key     *ecdsa.PrivateKey
		CertPEM []byte
		KeyPEM  []byte
	}
)

// NewCA creates a new self-signed CA certificate and key pair.
func NewCA(opts CAOptions) (*CA, error) {
	validFor := opts.ValidFor
	if validFor <= 0 {
		validFor = 5 * 365 * 24 * time.Hour
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("certgen: generate CA key: %w", err)
	}
	serial, err := newSerialNumber()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: opts.CommonName, Organization: orgOrDefault(opts.Org)},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(validFor),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("certgen: create CA certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("certgen: parse CA certificate: %w", err)
	}
	keyPEM, err := encodeECKey(key)
	if err != nil {
		return nil, err
	}

	return &CA{
		Cert:    cert,
		Key:     key,
		CertPEM: encodeCertPEM(der),
		KeyPEM:  keyPEM,
	}, nil
}

// NewLeafCert issues a certificate signed by ca.
func NewLeafCert(ca *CA, opts CertOptions) (*Cert, error) {
	validFor := opts.ValidFor
	if validFor <= 0 {
		validFor = 90 * 24 * time.Hour
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("certgen: generate leaf key: %w", err)
	}
	serial, err := newSerialNumber()
	if err != nil {
		return nil, err
	}

	extKeyUsage := x509.ExtKeyUsageClientAuth
	if opts.IsServer {
		extKeyUsage = x509.ExtKeyUsageServerAuth
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: opts.CommonName, Organization: orgOrDefault(opts.Org)},
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.Add(validFor),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{extKeyUsage},
		DNSNames:     opts.DNSNames,
		IPAddresses:  opts.IPAddresses,
		URIs:         opts.URIs,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, &key.PublicKey, ca.Key)
	if err != nil {
		return nil, fmt.Errorf("certgen: create leaf certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("certgen: parse leaf certificate: %w", err)
	}
	keyPEM, err := encodeECKey(key)
	if err != nil {
		return nil, err
	}

	return &Cert{
		Cert:    cert,
		Key:     key,
		CertPEM: encodeCertPEM(der),
		KeyPEM:  keyPEM,
	}, nil
}

func newSerialNumber() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("certgen: generate serial number: %w", err)
	}
	return serial, nil
}

func orgOrDefault(org string) []string {
	if org == "" {
		return nil
	}
	return []string{org}
}

func encodeCertPEM(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func encodeECKey(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("certgen: marshal EC private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), nil
}
