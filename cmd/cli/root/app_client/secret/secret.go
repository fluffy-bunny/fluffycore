// Package secret implements `cli app_client secret set|get`: direct gRPC
// calls to Greeter.SetSecret/GetSecret over mutual TLS, presenting a client
// certificate and no bearer token at all -- proof, end to end, that the
// mtls_verified claim path (see middleware/auth/mtls and
// example/internal/auth) is what let the call through. Deliberately bypasses
// the generated IAppGreeterClientAccessor (its dial options are hardcoded
// insecure -- fine for the other app_client commands, wrong for this one) and
// dials directly with the checked-in dev client certificate instead.
package secret

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	cobra_utils "github.com/fluffy-bunny/fluffycore/cmd/cli/internal/cobra_utils"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	proto_hellowworld_models "github.com/fluffy-bunny/fluffycore/proto/helloworld/models"
	cobra "github.com/spf13/cobra"
	grpc "google.golang.org/grpc"
	credentials "google.golang.org/grpc/credentials"
)

const use = "secret"

var printer = cobra_utils.NewPrinter()

var (
	addr       string
	serverName string
	certFile   string
	keyFile    string
	caFile     string
	orgID      string
	secretKey  string
	secretVal  string
)

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             "Call Greeter.SetSecret/GetSecret over mutual TLS (no JWT involved)",
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}
	command.PersistentFlags().StringVar(&addr, "addr", "localhost:50051", "Greeter gRPC server address")
	command.PersistentFlags().StringVar(&serverName, "server-name", "localhost", "expected TLS server name (must match a SAN on the server certificate)")
	command.PersistentFlags().StringVar(&certFile, "cert", "./example/server/certs/client-cert.pem", "client certificate presented for mutual TLS")
	command.PersistentFlags().StringVar(&keyFile, "cert-key", "./example/server/certs/client-key.pem", "private key for --cert")
	command.PersistentFlags().StringVar(&caFile, "ca", "./example/server/certs/ca.pem", "CA bundle the server's certificate must verify against")
	command.PersistentFlags().StringVar(&orgID, "org-id", "acme", "org_id field on the request")
	command.PersistentFlags().StringVar(&secretKey, "secret-key", "foo", "the secret's key")

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "SetSecret",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, conn, err := newClient()
			if err != nil {
				return err
			}
			defer conn.Close()

			resp, err := client.SetSecret(context.Background(), &proto_hellowworld_models.SetSecretRequest{
				OrgId: orgID,
				Key:   secretKey,
				Value: secretVal,
			})
			if err != nil {
				return fmt.Errorf("SetSecret: %w", err)
			}
			printer.Successf("SetSecret ok: success=%v (org_id=%s key=%s)", resp.Success, orgID, secretKey)
			return nil
		},
	}
	setCmd.Flags().StringVar(&secretVal, "secret-value", "bar", "the value to store")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "GetSecret",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, conn, err := newClient()
			if err != nil {
				return err
			}
			defer conn.Close()

			resp, err := client.GetSecret(context.Background(), &proto_hellowworld_models.GetSecretRequest{
				OrgId: orgID,
				Key:   secretKey,
			})
			if err != nil {
				return fmt.Errorf("GetSecret: %w", err)
			}
			printer.Successf("GetSecret ok: value=%q (org_id=%s key=%s)", resp.Value, orgID, secretKey)
			return nil
		},
	}

	command.AddCommand(setCmd)
	command.AddCommand(getCmd)
	parentCmd.AddCommand(command)
}

// newClient dials addr presenting the configured client certificate -- no
// bearer token, no Authorization header at all -- and returns a raw
// GreeterClient. The call only succeeds because SetSecret/GetSecret's
// entrypoint config accepts mtls_verified as an alternative to a JWT (see
// example/internal/auth.BuildGrpcEntrypointPermissionsClaimsMap); dropping
// --cert/--cert-key (or pointing them at a certificate the server's --ca
// doesn't trust) demonstrates the denied path instead.
func newClient() (proto_helloworld.GreeterClient, *grpc.ClientConn, error) {
	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("load client cert/key: %w", err)
	}
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, nil, fmt.Errorf("read CA file: %w", err)
	}
	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(caPEM) {
		return nil, nil, fmt.Errorf("CA file %s contains no usable certificates", caFile)
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      rootCAs,
		ServerName:   serverName,
	})
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return proto_helloworld.NewGreeterClient(conn), conn, nil
}
