package nats_connect

import (
	"encoding/json"
	"fmt"
)

type isNATSTokenConnectType interface {
	isNATSTokenConnectType()
}

type ClientCredentialsTokenType struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scopes       string `json:"scopes"`
	// the Account Audience
	Account string `json:"account"`
}

func (*ClientCredentialsTokenType) isNATSTokenConnectType() {}

type MastodonTokenType struct {
	AccessToken string `json:"access_token"`
}

func (*MastodonTokenType) isNATSTokenConnectType() {}

type NATSConnectToken struct {
	// Types that are assignable to Update:
	//
	//	*ClientCredentialsTokenType
	//	*MastodonTokenType
	Token isNATSTokenConnectType `json:"token"`
}

func (m *NATSConnectToken) GetToken() isNATSTokenConnectType {
	if m != nil {
		return m.Token
	}
	return nil
}
func (m *NATSConnectToken) MarshalJSON() ([]byte, error) {
	switch token := m.Token.(type) {
	case *ClientCredentialsTokenType:
		return json.Marshal(&struct {
			Token     *ClientCredentialsTokenType `json:"token"`
			TokenType string                      `json:"token_type"`
		}{Token: token, TokenType: "ClientCredentialsTokenType"})
	case *MastodonTokenType:
		return json.Marshal(&struct {
			Token     *MastodonTokenType `json:"token"`
			TokenType string             `json:"token_type"`
		}{Token: token, TokenType: "MastodonTokenType"})
	default:
		return nil, fmt.Errorf("unknown token type")
	}
}
func (m *NATSConnectToken) UnmarshalJSON(data []byte) error {
	// Temporary struct to determine the type
	temp := struct {
		TokenType string `json:"token_type"`
	}{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	switch temp.TokenType {
	case "ClientCredentialsTokenType":
		temp := struct {
			Token *ClientCredentialsTokenType `json:"token"`
		}{}
		if err := json.Unmarshal(data, &temp); err != nil {
			return err
		}
		m.Token = temp.Token
	case "MastodonTokenType":
		temp := struct {
			Token *MastodonTokenType `json:"token"`
		}{}
		if err := json.Unmarshal(data, &temp); err != nil {
			return err
		}
		m.Token = temp.Token
	default:
		return fmt.Errorf("unknown token type")
	}
	return nil
}
