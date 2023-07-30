package echo

type WellKnownOpenidConfiguration struct {
	Issuer                                     string   `json:"issuer,omitempty" mapstructure:"issuer"`
	JwksURI                                    string   `json:"jwks_uri,omitempty" mapstructure:"jwks_uri"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint,omitempty" mapstructure:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint,omitempty" mapstructure:"token_endpoint"`
	UserinfoEndpoint                           string   `json:"userinfo_endpoint,omitempty" mapstructure:"userinfo_endpoint"`
	EndSessionEndpoint                         string   `json:"end_session_endpoint,omitempty" mapstructure:"end_session_endpoint"`
	CheckSessionIframe                         string   `json:"check_session_iframe,omitempty" mapstructure:"check_session_iframe"`
	RevocationEndpoint                         string   `json:"revocation_endpoint,omitempty" mapstructure:"revocation_endpoint"`
	IntrospectionEndpoint                      string   `json:"introspection_endpoint,omitempty" mapstructure:"introspection_endpoint"`
	DeviceAuthorizationEndpoint                string   `json:"device_authorization_endpoint,omitempty" mapstructure:"device_authorization_endpoint"`
	FrontchannelLogoutSupported                bool     `json:"frontchannel_logout_supported,omitempty" mapstructure:"frontchannel_logout_supported"`
	FrontchannelLogoutSessionSupported         bool     `json:"frontchannel_logout_session_supported,omitempty" mapstructure:"frontchannel_logout_session_supported"`
	BackchannelLogoutSupported                 bool     `json:"backchannel_logout_supported,omitempty" mapstructure:"backchannel_logout_supported"`
	BackchannelLogoutSessionSupported          bool     `json:"backchannel_logout_session_supported,omitempty" mapstructure:"backchannel_logout_session_supported"`
	ScopesSupported                            []string `json:"scopes_supported,omitempty" mapstructure:"scopes_supported"`
	ClaimsSupported                            []string `json:"claims_supported,omitempty" mapstructure:"claims_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported,omitempty" mapstructure:"grant_types_supported"`
	ResponseTypesSupported                     []string `json:"response_types_supported,omitempty" mapstructure:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported,omitempty" mapstructure:"response_modes_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported,omitempty" mapstructure:"token_endpoint_auth_methods_supported"`
	SubjectTypesSupported                      []string `json:"subject_types_supported,omitempty" mapstructure:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported           []string `json:"id_token_signing_alg_values_supported,omitempty" mapstructure:"id_token_signing_alg_values_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported,omitempty" mapstructure:"code_challenge_methods_supported"`
	RequestParameterSupported                  bool     `json:"request_parameter_supported,omitempty" mapstructure:"request_parameter_supported"`
	RequestObjectSigningAlgValuesSupported     []string `json:"request_object_signing_alg_values_supported,omitempty" mapstructure:"request_object_signing_alg_values_supported"`
	AuthorizationResponseIssParameterSupported bool     `json:"authorization_response_iss_parameter_supported,omitempty" mapstructure:"authorization_response_iss_parameter_supported"`
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
