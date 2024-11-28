package nats_token

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOneOf(t *testing.T) {
	natsToken := &NATSConnectToken{
		Token: &ClientCredentialsTokenType{
			Scopes:       "scopes",
			ClientID:     "client_id",
			ClientSecret: "client_secret",
		},
	}

	tt := natsToken.GetToken()
	require.NotNil(t, tt)
	switch oneofToken := tt.(type) {
	case *ClientCredentialsTokenType:
		require.Equal(t, "client_id", oneofToken.ClientID)
		require.Equal(t, "client_secret", oneofToken.ClientSecret)
		require.Equal(t, "scopes", oneofToken.Scopes)
	default:
		require.Fail(t, "unexpected type")
	}

	jToken, _ := json.Marshal(natsToken)
	natsToken = &NATSConnectToken{}
	err := json.Unmarshal(jToken, natsToken)
	require.NoError(t, err)
	tt = natsToken.GetToken()

	switch oneofToken := tt.(type) {
	case *ClientCredentialsTokenType:
		require.Equal(t, "client_id", oneofToken.ClientID)
		require.Equal(t, "client_secret", oneofToken.ClientSecret)
		require.Equal(t, "scopes", oneofToken.Scopes)
	default:
		require.Fail(t, "unexpected type")
	}

	natsToken = &NATSConnectToken{
		Token: &MastodonTokenType{
			AccessToken: "access_token",
		},
	}

	tt = natsToken.GetToken()
	require.NotNil(t, tt)
	switch oneofToken := tt.(type) {
	case *MastodonTokenType:
		require.Equal(t, "access_token", oneofToken.AccessToken)
	default:
		require.Fail(t, "unexpected type")
	}

}
