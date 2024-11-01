package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	testservices "github.com/fluffy-bunny/fluffycore/mocks/testservices"
	jwt "github.com/golang-jwt/jwt/v4"
	echo "github.com/labstack/echo/v4"
	jwk "github.com/lestrrat-go/jwx/v2/jwk"
	jwxt "github.com/lestrrat-go/jwx/v2/jwt"
)

type (
	PublicJwk struct {
		Alg string `json:"alg"`
		Crv string `json:"crv"`
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Use string `json:"use"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}
	PrivateJwk struct {
		Alg string `json:"alg"`
		Crv string `json:"crv"`
		D   string `json:"d"`
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Use string `json:"use"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}
	SigningKey struct {
		PrivateKey string     `json:"private_key"`
		PublicKey  string     `json:"public_key"`
		NotBefore  time.Time  `json:"not_before"`
		NotAfter   time.Time  `json:"not_after"`
		Password   string     `json:"password"`
		Kid        string     `json:"kid"`
		PublicJwk  PublicJwk  `json:"public_jwk"`
		PrivateJwk PrivateJwk `json:"private_jwk"`
	}
	JWKSKeys struct {
		Keys []PublicJwk `json:"keys"`
	}
)

const jwtKey = `{
        "private_key": "-----BEGIN PRIVATE KEY-----\nMC4CAQAwBQYDK2VwBCIEIFyg95QloKek6oJQBWtJZL8u8ZDGOLjGsTp7ejUK/hUJ\n-----END PRIVATE KEY-----\n",
        "public_key": "-----BEGIN ED25519 PUBLIC KEY-----\nMCowBQYDK2VwAyEAonYSt2V0HhMZSpiu2Mw9xz75aSUf2jYH1Hwn2Xz173s=\n-----END ED25519 PUBLIC KEY-----\n",
        "not_before": "2023-11-01T14:50:22.7555184-07:00",
        "not_after": "2030-11-01T14:50:22.7555184-07:00",
        "kid": "526756287cc1938baa1a35c8b7a32368",
        "public_jwk": {
            "alg": "EdDSA",
            "crv": "Ed25519",
            "kid": "526756287cc1938baa1a35c8b7a32368",
            "kty": "OKP",
            "use": "sig",
            "x": "onYSt2V0HhMZSpiu2Mw9xz75aSUf2jYH1Hwn2Xz173s"
        },
        "private_jwk": {
            "alg": "EdDSA",
            "crv": "Ed25519",
            "kid": "526756287cc1938baa1a35c8b7a32368",
            "kty": "OKP",
            "use": "sig",
            "x": "onYSt2V0HhMZSpiu2Mw9xz75aSUf2jYH1Hwn2Xz173s",
            "d": "XKD3lCWgp6TqglAFa0lkvy7xkMY4uMaxOnt6NQr-FQk"
        }
    }`

var signingKey *SigningKey
var keySet jwk.Set
var jwksKeys *JWKSKeys

func init() {
	var _ IClaims = (*Claims)(nil)
	var _ jwt.Claims = (*Claims)(nil)
	signingKey, _ = LoadSigningKey()
	keySet = KeySet()
	jwksKeys = &JWKSKeys{
		Keys: []PublicJwk{
			signingKey.PublicJwk,
		},
	}

}
func KeySet() jwk.Set {
	keyB, _ := json.Marshal(signingKey.PrivateJwk)
	privkey, err := jwk.ParseKey(keyB)
	if err != nil {
		panic(err)
	}
	pubkey, err := jwk.PublicKeyOf(privkey)
	if err != nil {
		panic(err)
	}
	set := jwk.NewSet()
	set.AddKey(pubkey)

	return set
}
func LoadSigningKey() (*SigningKey, error) {
	var key SigningKey
	err := json.Unmarshal([]byte(jwtKey), &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func NewOauth2MockServer() *httptest.Server {
	mockServer := testservices.NewMockServer(nil, newMockServerProcedures()...)
	return mockServer
}
func newMockServerProcedures() []testservices.MockServerProcedure {
	var procedures []testservices.MockServerProcedure
	procedures = append(procedures, testservices.MockServerProcedure{
		URI:        "/.well-known/openid-configuration",
		HTTPMethod: http.MethodGet,
		Response: testservices.MockResponse{
			StatusCode: http.StatusOK,
			BodyFunc: func(r *http.Request) ([]byte, int) {
				baseUrl := "http://" + r.Host
				wellKnownOpenidConfigurationResponse := WellKnownOpenidConfiguration{
					Issuer:                             baseUrl,
					JwksURI:                            baseUrl + "/.well-known/jwks",
					TokenEndpoint:                      baseUrl + "/oauth/token",
					FrontchannelLogoutSupported:        true,
					FrontchannelLogoutSessionSupported: true,
					BackchannelLogoutSupported:         true,
					BackchannelLogoutSessionSupported:  true,
					ScopesSupported:                    []string{"offline_access"},
					ClaimsSupported:                    []string{},
					GrantTypesSupported: []string{
						"authorization_code",
						"client_credentials",
					},
					ResponseTypesSupported: []string{
						"code",
						"token",
						"id_token",
						"id_token token",
						"code id_token",
						"code token",
						"code id_token token",
					},
					ResponseModesSupported: []string{
						"form_post",
						"query",
						"fragment",
					},
					TokenEndpointAuthMethodsSupported: []string{
						"client_secret_basic",
						"client_secret_post",
					},
					SubjectTypesSupported:            []string{"public"},
					IDTokenSigningAlgValuesSupported: []string{"EdDSA"},
					CodeChallengeMethodsSupported: []string{"plain",
						"S256"},
					RequestParameterSupported: true,
					RequestObjectSigningAlgValuesSupported: []string{
						"EdDSA",
						"RS256",
						"RS384",
						"RS512",
						"PS256",
						"PS384",
						"PS512",
						"ES256",
						"ES384",
						"ES512",
						"HS256",
						"HS384",
						"HS512"},
					AuthorizationResponseIssParameterSupported: true,
				}
				result, _ := json.Marshal(&wellKnownOpenidConfigurationResponse)
				return result, http.StatusOK
			},
		},
	})

	procedures = append(procedures, testservices.MockServerProcedure{
		URI:        "/.well-known/jwks",
		HTTPMethod: http.MethodGet,
		RequestValidator: func(r *http.Request) int {
			return http.StatusOK
		},
		Response: testservices.MockResponse{
			StatusCode: http.StatusOK,
			BodyFunc: func(r *http.Request) ([]byte, int) {
				data, err := json.Marshal(jwksKeys)
				if err != nil {
					return nil, http.StatusInternalServerError
				}
				return data, http.StatusOK
			},
		},
	})
	procedures = append(procedures, testservices.MockServerProcedure{
		URI:              "/oauth/token",
		HTTPMethod:       http.MethodPost,
		RequestValidator: tokenEndpointRequestValidator,
		Response: testservices.MockResponse{
			StatusCode: http.StatusOK,
			BodyFunc:   tokenEndpointRequestHandler,
		},
	})
	return procedures
}
func tokenEndpointRequestValidator(r *http.Request) int {
	req := &tokenRequest{}
	binder := &echo.DefaultBinder{}
	ctx := echo.New().NewContext(r, nil)
	binder.Bind(req, ctx)

	switch req.GrantType {
	case "client_credentials":
		return http.StatusOK
	case "refresh_token":
		if len(req.RefreshToken) > 0 {
			return http.StatusOK
		}
	}

	return http.StatusBadRequest
}

type tokenRequest struct {
	ClientID     string `json:"client_id" form:"client_id" query:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret" query:"client_secret"`
	GrantType    string `json:"grant_type" form:"grant_type" query:"grant_type"`
	RefreshToken string `json:"refresh_token" form:"refresh_token" query:"refresh_token"`
}

var tokenJson = `{
	"aud": [
		"fluffy-micro"
	],
	"client_id": "fluffy-micro",
	"exp": 1690418966,
	"iat": 1690415366,
	"iss": "http://fluffy",
	"jti": "cj0r21jdphac7388vqmg",
	"permissions": [
		"Storage.Read.All",
		"Storage.ReadWrite.All"
	]
}`

func tokenEndpointRequestHandler(r *http.Request) ([]byte, int) {
	req := &tokenRequest{}
	binder := &echo.DefaultBinder{}
	ctx := echo.New().NewContext(r, nil)
	binder.Bind(req, ctx)

	switch req.GrantType {
	case "client_credentials":
		tokenParts := map[string]interface{}{}
		err := json.Unmarshal([]byte(tokenJson), &tokenParts)
		if err != nil {
			return nil, http.StatusInternalServerError
		}
		var expSeconds int = 3600
		tokenParts["iat"] = time.Now().Unix()
		tokenParts["exp"] = time.Now().Add(time.Second * time.Duration(expSeconds)).Unix()
		claims := NewClaims()
		for k, v := range tokenParts {
			claims.Set(k, v)
		}
		claims.Set("client_id", req.ClientID)
		token, err := MintToken(claims)
		if err != nil {
			return nil, http.StatusInternalServerError
		}
		response := ClientCredentialsTokenResponse{
			AccessToken: token,
			ExpiresIn:   expSeconds,
			TokenType:   "Bearer",
		}
		resBytes, _ := json.Marshal(&response)
		return resBytes, http.StatusOK
	case "refresh_token":
		if req.RefreshToken == "1234" {
			response := RefreshTokenResponse{
				RefreshToken: "1234",
				AccessToken:  "abcd",
				ExpiresIn:    300,
				TokenType:    "Bearer",
				Scope:        "scope1 offline_access",
			}
			resBytes, _ := json.Marshal(&response)
			return resBytes, http.StatusOK
		}
	}

	return nil, http.StatusBadRequest
}

type (
	Claims  map[string]interface{}
	IClaims interface {
		Valid() error
		Set(key string, value interface{}) error
		Delete(key string) error
		Get(key string) interface{}
		JwtClaims() jwt.Claims
		Claims() Claims
	}
)

func NewClaims() IClaims {
	return &Claims{}
}

// Valid claims verification
func (a *Claims) Valid() error {
	return nil
}
func (a *Claims) Claims() Claims {
	return *a
}

func (a *Claims) Set(key string, value interface{}) error {
	(*a)[key] = value
	return nil
}
func (a *Claims) Delete(key string) error {
	delete(*a, key)
	return nil
}
func (a *Claims) Get(key string) interface{} {
	return (*a)[key]
}
func (a *Claims) JwtClaims() jwt.Claims {
	return a
}

func ParseTokenRaw(accessToken string) (jwxt.Token, error) {
	// Parse the JWT
	parseOptions := []jwxt.ParseOption{}

	parseOptions = append(parseOptions, jwxt.WithKeySet(keySet))

	token, err := jwxt.ParseString(accessToken, parseOptions...)
	if err != nil {
		return nil, err
	}

	// This set had a key that worked
	var validationOpts []jwxt.ValidateOption
	//validationOpts = append(validationOpts, jwxt.WithIssuer(s.Options.OAuth2Document.Issuer))

	// Allow clock skew
	validationOpts = append(validationOpts,
		jwxt.WithAcceptableSkew(time.Minute*time.Duration(5)))

	opts := validationOpts
	err = jwxt.Validate(token, opts...)
	if err != nil {
		return nil, err
	}
	return token, nil
}
func ValidateToken(ctx context.Context, accessToken string) (IClaims, error) {
	token, err := ParseTokenRaw(accessToken)
	if err != nil {
		return nil, err
	}
	claimMap, err := token.AsMap(ctx)
	if err != nil {
		return nil, err
	}

	claims := NewClaims()
	for k, v := range claimMap {
		claims.Set(k, v)
	}
	return claims, nil
}
func MintToken(claims IClaims) (string, error) {
	var method jwt.SigningMethod
	switch signingKey.PrivateJwk.Alg {
	case "RS256":
		method = jwt.SigningMethodRS256
	case "RS384":
		method = jwt.SigningMethodRS384
	case "RS512":
		method = jwt.SigningMethodRS512
	case "ES256":
		method = jwt.SigningMethodES256
	case "ES384":
		method = jwt.SigningMethodES384
	case "ES512":
		method = jwt.SigningMethodES512
	case "EdDSA":
		method = jwt.SigningMethodEdDSA
	default:
		return "", fmt.Errorf("unsupported signing method: %s", signingKey.PrivateJwk.Alg)
	}
	kid := signingKey.Kid
	signedKey := []byte(signingKey.PrivateKey)

	var getKey = func() (interface{}, error) {
		var key interface{}

		if strings.HasPrefix(signingKey.PrivateJwk.Alg, "Ed") {
			v, err := jwt.ParseEdPrivateKeyFromPEM(signedKey)
			if err != nil {
				return "", err
			}
			key = v
			return key, nil
		}

		if strings.HasPrefix(signingKey.PrivateJwk.Alg, "ES") {
			v, err := jwt.ParseECPrivateKeyFromPEM(signedKey)
			if err != nil {
				return "", err
			}
			key = v
			return key, nil
		}

		v, err := jwt.ParseRSAPrivateKeyFromPEM(signedKey)
		if err != nil {
			return "", err
		}
		key = v
		return key, nil
	}
	token := jwt.NewWithClaims(method, claims.JwtClaims())
	token.Header["kid"] = kid
	key, err := getKey()
	if err != nil {
		return "", err
	}

	// special case, aud is allowed

	jwtToken, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return jwtToken, nil
}
