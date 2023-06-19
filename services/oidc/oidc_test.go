package oidc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoogleDiscovery(t *testing.T) {
	t.Skip("skipping test")
	ctx := context.Background()

	openIdConnectConfiguration, err := New(ctx, "https://accounts.google.com")
	require.NoError(t, err)
	require.NotNil(t, openIdConnectConfiguration)
	discoveryDocument, err := openIdConnectConfiguration.GetDiscoveryDocument(ctx)
	require.NoError(t, err)
	require.NotNil(t, discoveryDocument)
	require.Equal(t, "https://accounts.google.com", discoveryDocument.Issuer)

}
