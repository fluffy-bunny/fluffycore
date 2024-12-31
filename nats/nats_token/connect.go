package nats_token

import (
	"encoding/base64"
	"encoding/json"

	status "github.com/gogo/status"
	nats "github.com/nats-io/nats.go"
	codes "google.golang.org/grpc/codes"
)

type (
	NATSConnectTokenClientCredentialsRequest struct {
		NATSUrl      string `json:"nats_url"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client"`
		Account      string `json:"account"`
	}
	CreateNATSConnectTokenClientCredentialsRequest struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client"`
		Account      string `json:"account"`
	}
)

func CreateNATSConnectTokenClientCredentials(request *CreateNATSConnectTokenClientCredentialsRequest) (string, error) {
	natsConnectToken := &NATSConnectToken{
		Token: &ClientCredentialsTokenType{
			ClientID:     request.ClientID,
			ClientSecret: request.ClientSecret,
			Account:      request.Account,
		},
	}
	natsConnectTokenJson, _ := json.Marshal(natsConnectToken)
	encodedToken := base64.StdEncoding.EncodeToString(natsConnectTokenJson)

	return encodedToken, nil
}

func CreateNatsConnectionWithClientCredentials(request *NATSConnectTokenClientCredentialsRequest) (*nats.Conn, error) {
	token, _ := CreateNATSConnectTokenClientCredentials(&CreateNATSConnectTokenClientCredentialsRequest{
		ClientID:     request.ClientID,
		ClientSecret: request.ClientSecret,
		Account:      request.Account,
	})
	tokenHandler := nats.TokenHandler(func() string {
		return token
	})
	nc, err := nats.Connect(request.NATSUrl, tokenHandler)
	return nc, err
}

func DecodeNATSConnectTokenClientCredentials(token string) (*ClientCredentialsTokenType, error) {
	natsConnectTokenJson, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	natsConnectToken := &NATSConnectToken{}
	err = json.Unmarshal([]byte(natsConnectTokenJson), natsConnectToken)
	if err != nil {
		return nil, err
	}
	rr, ok := natsConnectToken.Token.(*ClientCredentialsTokenType)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid token type")
	}
	return rr, nil
}
