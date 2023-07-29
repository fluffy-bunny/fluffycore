package echo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	echo "github.com/labstack/echo/v4"
	jwk "github.com/lestrrat-go/jwx/v2/jwk"
)

const jwtKey = `{
    "private_key": "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIFA+8y3M5qxkjuI7HOUAPVgrsjUnu5kwRjsZlbCmyabCoAoGCCqGSM49\nAwEHoUQDQgAEYMrUm/S5+d+euQHrrzQMWJSFafSYcgUE0RYjfI7sErK75hSdE0aj\nPNMXaaDG395zD18VxjsmqPTWom17ncVnnw==\n-----END EC PRIVATE KEY-----\n",
    "public_key": "-----BEGIN EC  PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEYMrUm/S5+d+euQHrrzQMWJSFafSY\ncgUE0RYjfI7sErK75hSdE0ajPNMXaaDG395zD18VxjsmqPTWom17ncVnnw==\n-----END EC  PUBLIC KEY-----\n",
    "not_before": "2022-01-02T00:00:00Z",
    "not_after": "2023-01-02T00:00:00Z",
    "password": "",
    "kid": "0b2cd2e54c924ce89f010f242862367d",
    "public_jwk": {
        "alg": "ES256",
        "crv": "P-256",
        "kid": "0b2cd2e54c924ce89f010f242862367d",
        "kty": "EC",
        "use": "sig",
        "x": "YMrUm_S5-d-euQHrrzQMWJSFafSYcgUE0RYjfI7sErI",
        "y": "u-YUnRNGozzTF2mgxt_ecw9fFcY7Jqj01qJte53FZ58"
    },
    "private_jwk": {
        "alg": "ES256",
        "crv": "P-256",
        "d": "UD7zLczmrGSO4jsc5QA9WCuyNSe7mTBGOxmVsKbJpsI",
        "kid": "0b2cd2e54c924ce89f010f242862367d",
        "kty": "EC",
        "use": "sig",
        "x": "YMrUm_S5-d-euQHrrzQMWJSFafSYcgUE0RYjfI7sErI",
        "y": "u-YUnRNGozzTF2mgxt_ecw9fFcY7Jqj01qJte53FZ58"
    }
}`

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

func NewOAuth2TestServer() *echo.Echo {
	e := echo.New()
	e.GET("/.well-known/jwks", func(c echo.Context) error {
		return c.JSON(http.StatusOK, jwksKeys)
	})
	e.GET("/.well-known/openid-configuration", func(c echo.Context) error {
		baseUrl := "http://" + c.Request().Host
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
			IDTokenSigningAlgValuesSupported: []string{"ES256"},
			CodeChallengeMethodsSupported: []string{"plain",
				"S256"},
			RequestParameterSupported: true,
			RequestObjectSigningAlgValuesSupported: []string{"RS256",
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
		return c.JSON(http.StatusOK, wellKnownOpenidConfigurationResponse)
	})

	return e
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
	default:
		return "", fmt.Errorf("unsupported signing method: %s", signingKey.PrivateJwk.Alg)
	}
	kid := signingKey.Kid
	signedKey := []byte(signingKey.PrivateKey)

	var getKey = func() (interface{}, error) {
		var key interface{}
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
