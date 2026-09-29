package otel

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_config "github.com/fluffy-bunny/fluffycore/contracts/config"
	fluffycore_contracts_otel "github.com/fluffy-bunny/fluffycore/contracts/otel"
	certgen "github.com/fluffy-bunny/fluffycore/utils/certgen"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// The runtime dials its own server (gateway, healthz, NATS bridge) with TLS exactly when
// this reports true.
func TestServerTLSEnabled_ReflectsConfigureServerOpts(t *testing.T) {
	build := func(core *fluffycore_contracts_config.CoreConfig) *FluffyCoreOTELStartup {
		b := di.Builder()
		di.AddInstance[*fluffycore_contracts_config.CoreConfig](b, core)
		s := NewFluffyCoreOTELStartup(&FluffyCoreOTELStartupConfig{
			FuncAuthGetEntryPointConfigs: func() map[string]contracts_common.IEntryPointConfig {
				return map[string]contracts_common.IEntryPointConfig{}
			},
		})
		s.SetConfig(&fluffycore_contracts_otel.OTELConfig{ServiceName: "tls-flag"})
		s.SetRootContainer(b.Build())
		_ = s.ConfigureServerOpts(zerolog.Nop().WithContext(context.Background()))
		return s
	}

	require.False(t, build(&fluffycore_contracts_config.CoreConfig{}).ServerTLSEnabled())

	dir := t.TempDir()
	ca, err := certgen.NewCA(certgen.CAOptions{CommonName: "ca"})
	require.NoError(t, err)
	leaf, err := certgen.NewLeafCert(ca, certgen.CertOptions{CommonName: "localhost", DNSNames: []string{"localhost"}, IsServer: true})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "server-cert.pem"), leaf.CertPEM, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "server-key.pem"), leaf.KeyPEM, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ca.pem"), ca.CertPEM, 0o600))
	require.True(t, build(&fluffycore_contracts_config.CoreConfig{TLSCertsDir: dir}).ServerTLSEnabled())
}
