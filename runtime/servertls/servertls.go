// Package servertls builds the *tls.Config used by the gRPC server from
// contracts/config.CoreConfig, wiring up mutual TLS (mTLS) when a client CA
// bundle is configured.
//
// The certificate, key, and client CA bundle are all re-read from disk
// whenever their mtime changes (checked at most once every few seconds, on the
// hot path of a TLS handshake), so an external secret-injection agent --
// HashiCorp Vault Agent, the Vault Secrets Operator/CSI provider,
// cert-manager, ... -- rotating those files on disk takes effect without
// restarting the process. This matters in particular for Vault's PKI secrets
// engine, which typically issues short-lived certificates.
package servertls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	"github.com/rs/zerolog/log"
)

// Client authentication modes accepted by CoreConfig.TLSClientAuth.
const (
	ClientAuthNone          = "none"
	ClientAuthRequest       = "request"
	ClientAuthVerifyIfGiven = "verify_if_given"
	ClientAuthRequire       = "require"
)

// defaultMinCheckInterval bounds how often a handshake re-stats the
// cert/key/CA files to check for rotation. A stat() call is cheap, but there's
// no reason to do it more than a few times a second under heavy connection
// churn.
const defaultMinCheckInterval = 5 * time.Second

// resolvedConfig is CoreConfig's TLS fields after applying the TLSCertsDir
// convenience shortcut.
type resolvedConfig struct {
	enabled      bool
	certFile     string
	keyFile      string
	clientCAFile string
	clientAuth   string
}

// resolveConfig fills in whichever of TLSCertFile/TLSKeyFile/TLSClientCAFile
// were left empty from TLSCertsDir (using the exact filenames `gencert`
// writes), and implies TLSEnabled when TLSCertsDir is set -- pointing at a
// certs directory is the on/off switch in that case. Explicit fields always
// win over the derived path.
func resolveConfig(cfg *fluffycore_contracts_config.CoreConfig) resolvedConfig {
	r := resolvedConfig{
		enabled:      cfg.TLSEnabled,
		certFile:     cfg.TLSCertFile,
		keyFile:      cfg.TLSKeyFile,
		clientCAFile: cfg.TLSClientCAFile,
		clientAuth:   cfg.TLSClientAuth,
	}
	if cfg.TLSCertsDir != "" {
		r.enabled = true
		if r.certFile == "" {
			r.certFile = filepath.Join(cfg.TLSCertsDir, "server-cert.pem")
		}
		if r.keyFile == "" {
			r.keyFile = filepath.Join(cfg.TLSCertsDir, "server-key.pem")
		}
		if r.clientCAFile == "" {
			r.clientCAFile = filepath.Join(cfg.TLSCertsDir, "ca.pem")
		}
	}
	return r
}

// IsMutualTLSEnabled reports whether cfg resolves to mutual TLS -- a client CA
// bundle configured, whether directly via TLSClientCAFile or derived from
// TLSCertsDir. Exported so callers (e.g. startup logging) can report the
// *resolved* state without duplicating -- or, worse, going stale against --
// the derivation logic in resolveConfig.
func IsMutualTLSEnabled(cfg *fluffycore_contracts_config.CoreConfig) bool {
	if cfg == nil {
		return false
	}
	return resolveConfig(cfg).clientCAFile != ""
}

// BuildServerTLSConfig builds a *tls.Config for the gRPC server from cfg, or
// returns (nil, nil) if TLS isn't enabled -- callers should skip
// grpc.Creds(...) entirely in that case and run the plaintext server exactly
// as before this feature existed. TLS is enabled by either cfg.TLSEnabled or
// cfg.TLSCertsDir being set (see resolveConfig).
func BuildServerTLSConfig(cfg *fluffycore_contracts_config.CoreConfig) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}
	resolved := resolveConfig(cfg)
	if !resolved.enabled {
		return nil, nil
	}
	if resolved.certFile == "" || resolved.keyFile == "" {
		return nil, fmt.Errorf("servertls: TLS is enabled but no cert/key were resolved -- set tlsCertFile/tlsKeyFile, or tlsCertsDir")
	}

	clientAuth, err := parseClientAuth(resolved.clientAuth, resolved.clientCAFile)
	if err != nil {
		return nil, err
	}

	reloading, err := newReloadingCertificate(resolved.certFile, resolved.keyFile, resolved.clientCAFile, defaultMinCheckInterval)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: reloading.GetCertificate,
		ClientAuth:     clientAuth,
	}
	if resolved.clientCAFile != "" {
		// ClientCAs must come from a per-handshake callback too, for the same
		// hot-reload reason as the certificate: GetConfigForClient takes
		// precedence over the static fields on tlsConfig for the whole handshake,
		// so the returned config must carry everything (cloned from tlsConfig)
		// with ClientCAs refreshed and GetConfigForClient cleared to avoid
		// recursing back into itself.
		tlsConfig.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
			clone := tlsConfig.Clone()
			clone.ClientCAs = reloading.clientCAPool()
			clone.GetConfigForClient = nil
			return clone, nil
		}
	}
	return tlsConfig, nil
}

func parseClientAuth(mode, clientCAFile string) (tls.ClientAuthType, error) {
	if mode == "" {
		if clientCAFile != "" {
			return tls.VerifyClientCertIfGiven, nil
		}
		return tls.NoClientCert, nil
	}
	switch mode {
	case ClientAuthNone:
		return tls.NoClientCert, nil
	case ClientAuthRequest:
		return tls.RequestClientCert, nil
	case ClientAuthVerifyIfGiven:
		return tls.VerifyClientCertIfGiven, nil
	case ClientAuthRequire:
		return tls.RequireAndVerifyClientCert, nil
	default:
		return 0, fmt.Errorf("servertls: unknown tlsClientAuth %q (expected one of: %s, %s, %s, %s)",
			mode, ClientAuthNone, ClientAuthRequest, ClientAuthVerifyIfGiven, ClientAuthRequire)
	}
}

// reloadingCertificate re-reads the server cert/key -- and, when configured,
// the client CA bundle -- from disk whenever their mtimes change.
type reloadingCertificate struct {
	certFile, keyFile, clientCAFile string
	minCheckInterval                time.Duration

	mu          sync.RWMutex
	cert        *tls.Certificate
	clientCAs   *x509.CertPool
	certModTime time.Time
	keyModTime  time.Time
	caModTime   time.Time
	lastChecked time.Time
}

func newReloadingCertificate(certFile, keyFile, clientCAFile string, minCheckInterval time.Duration) (*reloadingCertificate, error) {
	r := &reloadingCertificate{
		certFile:         certFile,
		keyFile:          keyFile,
		clientCAFile:     clientCAFile,
		minCheckInterval: minCheckInterval,
	}
	if err := r.reload(true); err != nil {
		return nil, err
	}
	return r, nil
}

// reload re-reads the cert/key/CA files if force is true, or if the check
// interval has elapsed and any of their mtimes changed since the last
// successful load. It never clears an already-loaded certificate on error --
// callers keep serving the last-known-good cert rather than failing every
// handshake because of a transient read (e.g. mid-write by the
// secret-injection agent).
func (r *reloadingCertificate) reload(force bool) error {
	if !force {
		r.mu.RLock()
		due := time.Since(r.lastChecked) >= r.minCheckInterval
		r.mu.RUnlock()
		if !due {
			return nil
		}
	}

	certInfo, err := os.Stat(r.certFile)
	if err != nil {
		return fmt.Errorf("servertls: stat cert file: %w", err)
	}
	keyInfo, err := os.Stat(r.keyFile)
	if err != nil {
		return fmt.Errorf("servertls: stat key file: %w", err)
	}
	var caInfo os.FileInfo
	if r.clientCAFile != "" {
		caInfo, err = os.Stat(r.clientCAFile)
		if err != nil {
			return fmt.Errorf("servertls: stat client CA file: %w", err)
		}
	}

	r.mu.RLock()
	unchanged := r.cert != nil &&
		certInfo.ModTime().Equal(r.certModTime) &&
		keyInfo.ModTime().Equal(r.keyModTime) &&
		(r.clientCAFile == "" || caInfo.ModTime().Equal(r.caModTime))
	r.mu.RUnlock()
	if unchanged {
		r.mu.Lock()
		r.lastChecked = time.Now()
		r.mu.Unlock()
		return nil
	}

	cert, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return fmt.Errorf("servertls: load server key pair: %w", err)
	}

	var pool *x509.CertPool
	if r.clientCAFile != "" {
		caPEM, err := os.ReadFile(r.clientCAFile)
		if err != nil {
			return fmt.Errorf("servertls: read client CA file: %w", err)
		}
		pool = x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return fmt.Errorf("servertls: client CA file %s contains no usable certificates", r.clientCAFile)
		}
	}

	r.mu.Lock()
	reloaded := r.cert != nil // false only on the very first load
	r.cert = &cert
	r.clientCAs = pool
	r.certModTime = certInfo.ModTime()
	r.keyModTime = keyInfo.ModTime()
	if caInfo != nil {
		r.caModTime = caInfo.ModTime()
	}
	r.lastChecked = time.Now()
	r.mu.Unlock()

	if reloaded {
		log.Info().
			Str("certFile", r.certFile).
			Str("clientCAFile", r.clientCAFile).
			Msg("servertls: reloaded server certificate from disk")
	}
	return nil
}

// GetCertificate implements tls.Config.GetCertificate.
func (r *reloadingCertificate) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	if err := r.reload(false); err != nil {
		log.Error().Err(err).Msg("servertls: failed to check/reload server certificate, serving last-known-good")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cert, nil
}

// clientCAPool returns the current client CA pool, reloading first if due.
func (r *reloadingCertificate) clientCAPool() *x509.CertPool {
	if err := r.reload(false); err != nil {
		log.Error().Err(err).Msg("servertls: failed to check/reload client CA bundle, serving last-known-good")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clientCAs
}
