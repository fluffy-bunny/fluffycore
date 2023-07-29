package oauth2

type (
	Client struct {
		ClientID     string                 `json:"client_id"`
		ClientSecret string                 `json:"client_secret"`
		Claims       map[string]interface{} `json:"claims"`
		// Expiration is in seconds, can be negative to account for clock skew
		Expiration int `json:"expiration"`
		// Issuer is the issuer of the token, if nil then the host is used
		// use this to issuer a token that your service doesn't know about
		Issuer string `json:"issuer"`
	}
	MockOAuth2Config struct {
		Clients []Client `json:"clients"`
	}
)
