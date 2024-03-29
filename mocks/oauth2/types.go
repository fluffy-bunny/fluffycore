package oauth2

type WellKnownOpenidConfiguration struct {
	Issuer                                     string   `json:"issuer" mapstructure:"issuer"`
	JwksURI                                    string   `json:"jwks_uri" mapstructure:"jwks_uri"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint" mapstructure:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint" mapstructure:"token_endpoint"`
	UserinfoEndpoint                           string   `json:"userinfo_endpoint" mapstructure:"userinfo_endpoint"`
	EndSessionEndpoint                         string   `json:"end_session_endpoint" mapstructure:"end_session_endpoint"`
	CheckSessionIframe                         string   `json:"check_session_iframe" mapstructure:"check_session_iframe"`
	RevocationEndpoint                         string   `json:"revocation_endpoint" mapstructure:"revocation_endpoint"`
	IntrospectionEndpoint                      string   `json:"introspection_endpoint" mapstructure:"introspection_endpoint"`
	DeviceAuthorizationEndpoint                string   `json:"device_authorization_endpoint" mapstructure:"device_authorization_endpoint"`
	FrontchannelLogoutSupported                bool     `json:"frontchannel_logout_supported" mapstructure:"frontchannel_logout_supported"`
	FrontchannelLogoutSessionSupported         bool     `json:"frontchannel_logout_session_supported" mapstructure:"frontchannel_logout_session_supported"`
	BackchannelLogoutSupported                 bool     `json:"backchannel_logout_supported" mapstructure:"backchannel_logout_supported"`
	BackchannelLogoutSessionSupported          bool     `json:"backchannel_logout_session_supported" mapstructure:"backchannel_logout_session_supported"`
	ScopesSupported                            []string `json:"scopes_supported" mapstructure:"scopes_supported"`
	ClaimsSupported                            []string `json:"claims_supported" mapstructure:"claims_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported" mapstructure:"grant_types_supported"`
	ResponseTypesSupported                     []string `json:"response_types_supported" mapstructure:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported" mapstructure:"response_modes_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported" mapstructure:"token_endpoint_auth_methods_supported"`
	SubjectTypesSupported                      []string `json:"subject_types_supported" mapstructure:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported           []string `json:"id_token_signing_alg_values_supported" mapstructure:"id_token_signing_alg_values_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported" mapstructure:"code_challenge_methods_supported"`
	RequestParameterSupported                  bool     `json:"request_parameter_supported" mapstructure:"request_parameter_supported"`
	RequestObjectSigningAlgValuesSupported     []string `json:"request_object_signing_alg_values_supported" mapstructure:"request_object_signing_alg_values_supported"`
	AuthorizationResponseIssParameterSupported bool     `json:"authorization_response_iss_parameter_supported" mapstructure:"authorization_response_iss_parameter_supported"`
}
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token" mapstructure:"access_token"`
	ExpiresIn    int    `json:"expires_in" mapstructure:"expires_in"`
	TokenType    string `json:"token_type" mapstructure:"token_type"`
	RefreshToken string `json:"refresh_token" mapstructure:"refresh_token"`
	Scope        string `json:"scope" mapstructure:"scope"`
}
type ClientCredentialsTokenResponse struct {
	AccessToken string `json:"access_token" mapstructure:"access_token"`
	ExpiresIn   int    `json:"expires_in" mapstructure:"expires_in"`
	TokenType   string `json:"token_type" mapstructure:"token_type"`
}
