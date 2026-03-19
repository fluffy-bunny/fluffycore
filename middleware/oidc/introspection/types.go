package introspection

import "github.com/sirupsen/logrus"

type IntrospectionValidationOptions struct {
	Logger *logrus.Logger `json:"-"`

	IntrospectionURL string `json:"introspection_url"`

	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	Token string `json:"token"`
}
