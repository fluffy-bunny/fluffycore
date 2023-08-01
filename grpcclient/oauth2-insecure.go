package grpcclient

import (
	"context"

	"github.com/rs/zerolog/log"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"
)

type OAuth2Insecure struct {
	token string
}

// oauthAccess supplies PerRPCCredentials from a given token.
type oauthAccess struct {
	token oauth2.Token

	// sidecarSecured is set to true when a sidecar (like Envoy) handles TLS
	// for gRPC calls without us knowing...
	sidecarSecured bool
}

// NewOauthAccess constructs the PerRPCCredentials using a given token.
func NewOauthAccess(token *oauth2.Token, sidecarTLS bool) credentials.PerRPCCredentials {
	return oauthAccess{token: *token, sidecarSecured: sidecarTLS}
}

func (oa oauthAccess) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	// If a sidecar service mesh is not in use, make sure we are on a secure transport
	if !oa.sidecarSecured {
		ri, _ := credentials.RequestInfoFromContext(ctx)
		if err := credentials.CheckSecurityLevel(ri.AuthInfo, credentials.PrivacyAndIntegrity); err != nil {
			log.Warn().Msg("SENDING AUTHENTICATION IN THE CLEAR (NON-TLS) - DO NOT USE IN PRODUCTION")
		}
	}

	return map[string]string{
		"authorization": oa.token.Type() + " " + oa.token.AccessToken,
	}, nil
}

func (oa oauthAccess) RequireTransportSecurity() bool {
	return false
}

// oauthTokenSourceAccess supplies PerRPCCredentials from a given token.
type tokenSourceAccess struct {
	tokenSource oauth2.TokenSource

	// sidecarSecured is set to true when a sidecar (like Envoy) handles TLS
	// for gRPC calls without us knowing...
	sidecarSecured bool
}

// NewOauthAccessFromTokenSource constructs the PerRPCCredentials using a given token.
func NewOauthAccessFromTokenSource(tokenSource oauth2.TokenSource, sidecarTLS bool) credentials.PerRPCCredentials {
	return tokenSourceAccess{tokenSource: tokenSource, sidecarSecured: sidecarTLS}
}

func (oa tokenSourceAccess) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	// If a sidecar service mesh is not in use, make sure we are on a secure transport
	if !oa.sidecarSecured {
		ri, _ := credentials.RequestInfoFromContext(ctx)
		if err := credentials.CheckSecurityLevel(ri.AuthInfo, credentials.PrivacyAndIntegrity); err != nil {
			log.Warn().Msg("SENDING AUTHENTICATION IN THE CLEAR (NON-TLS) - DO NOT USE IN PRODUCTION")
		}
	}
	token, err := oa.tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": token.Type() + " " + token.AccessToken,
	}, nil

}

func (oa tokenSourceAccess) RequireTransportSecurity() bool {
	return false
}
