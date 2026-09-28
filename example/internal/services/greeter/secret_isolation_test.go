package greeter

import (
	"context"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	fluffycore_contracts_secretstore "github.com/fluffy-bunny/fluffycore/example/internal/contracts/secretstore"
	secretstore "github.com/fluffy-bunny/fluffycore/example/internal/services/secretstore"
	proto_hellowworld_models "github.com/fluffy-bunny/fluffycore/proto/helloworld/models"
	services_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	status "github.com/gogo/status"
	"github.com/stretchr/testify/require"
	codes "google.golang.org/grpc/codes"
)

func greeterFor(store fluffycore_contracts_secretstore.ISecretStore, claims ...fluffycore_contracts_common.Claim) *service {
	p := services_claimsprincipal.NewIClaimsPrincipal()
	p.AddClaim(claims...)
	return &service{secretStore: store, claimsPrincipal: p}
}

// OrgId comes from the request, so it used to be the only partition: any authenticated
// caller could read or overwrite any org's secrets by naming it.
func TestSecrets_IsolatedByAuthenticatedCaller(t *testing.T) {
	b := di.Builder()
	secretstore.AddSingletonSecretStore(b)
	store := di.Get[fluffycore_contracts_secretstore.ISecretStore](b.Build())
	ctx := context.Background()

	alice := greeterFor(store, fluffycore_contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeSub, Value: "alice"})
	mallory := greeterFor(store, fluffycore_contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeSub, Value: "mallory"})
	service := greeterFor(store,
		fluffycore_contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeSub, Value: fluffycore_wellknown.AnonymousSubject},
		fluffycore_contracts_common.Claim{Type: fluffycore_wellknown.ClaimTypeMTLSFingerprint, Value: "ab12"})

	_, err := alice.SetSecret(ctx, &proto_hellowworld_models.SetSecretRequest{OrgId: "acme", Key: "db", Value: "s3cret"})
	require.NoError(t, err)

	got, err := alice.GetSecret(ctx, &proto_hellowworld_models.GetSecretRequest{OrgId: "acme", Key: "db"})
	require.NoError(t, err)
	require.Equal(t, "s3cret", got.Value)

	_, err = mallory.GetSecret(ctx, &proto_hellowworld_models.GetSecretRequest{OrgId: "acme", Key: "db"})
	require.Equal(t, codes.NotFound, status.Code(err), "another caller naming the same org must not see it")
	_, err = service.GetSecret(ctx, &proto_hellowworld_models.GetSecretRequest{OrgId: "acme", Key: "db"})
	require.Equal(t, codes.NotFound, status.Code(err))

	// mTLS callers get their own partition, keyed by certificate fingerprint.
	_, err = service.SetSecret(ctx, &proto_hellowworld_models.SetSecretRequest{OrgId: "acme", Key: "db", Value: "svc"})
	require.NoError(t, err)
	got, err = alice.GetSecret(ctx, &proto_hellowworld_models.GetSecretRequest{OrgId: "acme", Key: "db"})
	require.NoError(t, err)
	require.Equal(t, "s3cret", got.Value, "mallory/service writes must not overwrite alice's secret")

	_, err = greeterFor(store).GetSecret(ctx, &proto_hellowworld_models.GetSecretRequest{OrgId: "acme", Key: "db"})
	require.Equal(t, codes.PermissionDenied, status.Code(err), "no identity at all")
}
