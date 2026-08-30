// Package gencert implements the `cli gencert` command: generates a local CA
// plus a server certificate and a client certificate signed by it, so mutual
// TLS can be exercised end-to-end (a real client-to-server handshake) without
// waiting on a real CA. Development/testing tool only -- in production, point
// tlsCertFile/tlsKeyFile/tlsClientCAFile at whatever your real certificate
// issuer (e.g. HashiCorp Vault's PKI secrets engine) injects onto disk
// instead. Shares its certificate/bundle generation with
// cobracore/cmd/gencert via utils/certgen -- this file is just the cobra
// wiring for this repo's own cmd/cli tool.
package gencert

import (
	"fmt"
	"path/filepath"
	"time"

	cobra_utils "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/cobra_utils"
	"github.com/fluffy-bunny/fluffycore/utils/certgen"
	cobra "github.com/spf13/cobra"
)

var (
	outDir     string
	hosts      []string
	serverCN   string
	clientCN   string
	clientURIs []string
	org        string
	validDays  int
	forever    bool
	force      bool
)

const use = "gencert"

var printer = cobra_utils.NewPrinter()

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             "Generate a local CA + server + client certificate set for testing mutual TLS",
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
	command.Flags().StringVarP(&outDir, "out", "o", "./certs", "output directory for the generated PEM files")
	command.Flags().StringArrayVarP(&hosts, "host", "H", []string{"localhost", "127.0.0.1"}, "SAN(s) for the server certificate (repeatable); IPs and DNS names are both accepted")
	command.Flags().StringVar(&serverCN, "server-cn", "localhost", "server certificate CommonName")
	command.Flags().StringVar(&clientCN, "client-cn", "fluffycore-test-client", "client certificate CommonName")
	command.Flags().StringArrayVar(&clientURIs, "client-uri", nil, "URI SAN(s) for the client certificate (repeatable), e.g. a SPIFFE ID such as spiffe://cluster.local/ns/foo/sa/bar")
	command.Flags().StringVar(&org, "org", "fluffycore-dev", "Organization field on every generated certificate")
	command.Flags().IntVar(&validDays, "days", 365, "validity period, in days, for the generated leaf certificates (the CA is valid 5x as long); ignored if --forever is set")
	command.Flags().BoolVar(&forever, "forever", false, "generate a certificate set that effectively never expires (100 years) -- for a checked-in dev bundle that shouldn't need periodic regeneration; never use for production certificates")
	command.Flags().BoolVar(&force, "force", false, "overwrite files already present in --out")
	parentCmd.AddCommand(command)
}

func run() error {
	printer.EnableColors = true

	validFor := time.Duration(validDays) * 24 * time.Hour
	if forever {
		validFor = certgen.Forever
	}

	bundle, err := certgen.NewBundle(certgen.BundleOptions{
		ServerCommonName: serverCN,
		ServerHosts:      hosts,
		ClientCommonName: clientCN,
		ClientURIs:       clientURIs,
		Org:              org,
		ValidFor:         validFor,
	})
	if err != nil {
		return fmt.Errorf("gencert: %w", err)
	}
	if err := bundle.WriteFiles(outDir, force); err != nil {
		return fmt.Errorf("gencert: %w", err)
	}

	abs, _ := filepath.Abs(outDir)
	printer.Infof(`Generated a local CA, server certificate, and client certificate in %s:

  ca.pem            self-signed CA certificate (trust this to verify client certs)
  ca-key.pem        CA private key -- keep it if you want to issue more certs later; the running server never needs it
  server-cert.pem   server certificate (CN=%s, SAN=%v)
  server-key.pem    server private key
  client-cert.pem   client certificate (CN=%s)
  client-key.pem    client private key

To run a fluffycore server with mutual TLS against this set, either point
tlsCertsDir (in your config JSON, or CoreConfig directly) at the output
directory:

  tlsCertsDir: %s

or set the three paths individually:

  tlsEnabled: true
  tlsCertFile: %s/server-cert.pem
  tlsKeyFile: %s/server-key.pem
  tlsClientCAFile: %s/ca.pem
  tlsClientAuth: verify_if_given

Dial it from a test client presenting client-cert.pem/client-key.pem, trusting
ca.pem as the server's root -- see middleware/auth/mtls's README for a worked
Go client snippet and how to gate a specific method on the resulting
mtls_verified/mtls_cn claims.`, abs, serverCN, hosts, clientCN, abs, abs, abs, abs)
	return nil
}
