package echo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	mocks_contracts_oauth2 "github.com/fluffy-bunny/fluffycore/mocks/contracts/oauth2"
	jwt "github.com/golang-jwt/jwt/v4"
	echo "github.com/labstack/echo/v4"
	jwk "github.com/lestrrat-go/jwx/v2/jwk"
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

type MockOAuth2Service struct {
	*echo.Echo
	config *mocks_contracts_oauth2.MockOAuth2Config
}

func (s *MockOAuth2Service) GetEcho() *echo.Echo {
	return s.Echo
}
func (s *MockOAuth2Service) GetClient(clientID string, clientSecret string) (bool, *mocks_contracts_oauth2.Client) {
	for _, client := range s.config.Clients {
		if client.ClientID == clientID && client.ClientSecret == clientSecret {
			return true, &client
		}
	}
	return false, nil
}
func (s *MockOAuth2Service) tokenHandler(c echo.Context) error {
	// get basic auth
	clientID, clientSecret, ok := c.Request().BasicAuth()
	if !ok {
		// try form
		clientID = c.FormValue("client_id")
		clientSecret = c.FormValue("client_secret")
	}
	if clientID == "" || clientSecret == "" {
		return c.JSON(http.StatusBadRequest, "invalid client_id or client_secret")
	}
	// get client
	ok, client := s.GetClient(clientID, clientSecret)
	if !ok {
		return c.JSON(http.StatusBadRequest, "invalid client_id or client_secret")
	}
	claims := NewClaims()
	for k, v := range client.Claims {
		claims.Set(k, v)
	}
	claims.Set("iss", "http://"+c.Request().Host)
	if client.Issuer != "" {
		// override the default issuer
		claims.Set("iss", client.Issuer)
	}
	claims.Set("client_id", clientID)

	now := time.Now()
	claims.Set("iat", now.Unix())
	claims.Set("exp", now.Add(time.Duration(client.Expiration)*time.Second).Unix())
	expSeconds := client.Expiration
	if expSeconds <= 0 {
		// if we are asked to mint an expired token, push the iat back 2x the expiration
		claims.Set("iat", now.Add(time.Duration(expSeconds*2)*time.Second).Unix())
		expSeconds = expSeconds * -1
	}

	token, err := MintToken(claims)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	response := ClientCredentialsTokenResponse{
		AccessToken: token,
		ExpiresIn:   expSeconds,
		TokenType:   "Bearer",
	}
	return c.JSON(http.StatusOK, response)
}
func (s *MockOAuth2Service) jwks(c echo.Context) error {
	return c.JSON(http.StatusOK, jwksKeys)
}
func (s *MockOAuth2Service) discovery(c echo.Context) error {
	baseUrl := "http://" + c.Request().Host
	wellKnownOpenidConfigurationResponse := WellKnownOpenidConfiguration{
		Issuer:          baseUrl,
		JwksURI:         baseUrl + "/.well-known/jwks",
		TokenEndpoint:   baseUrl + "/oauth/token",
		ScopesSupported: []string{"offline_access"},
		ClaimsSupported: []string{},
		GrantTypesSupported: []string{
			"client_credentials",
		},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_basic",
			"client_secret_post",
		},
	}
	return c.JSON(http.StatusOK, wellKnownOpenidConfigurationResponse)
}
func NewOAuth2TestServer(config *mocks_contracts_oauth2.MockOAuth2Config) *MockOAuth2Service {
	mockOAuth2Service := &MockOAuth2Service{
		Echo:   echo.New(),
		config: config,
	}

	mockOAuth2Service.GET("/.well-known/jwks", mockOAuth2Service.jwks)
	mockOAuth2Service.GET("/.well-known/openid-configuration", mockOAuth2Service.discovery)
	mockOAuth2Service.POST("/oauth/token", mockOAuth2Service.tokenHandler)

	return mockOAuth2Service
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
