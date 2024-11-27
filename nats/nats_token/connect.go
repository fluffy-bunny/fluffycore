package nats_connect

import (
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
)

func CreateNATSConnectTokenClientCredentials(request *NATSConnectTokenClientCredentialsRequest) (string, error) {
	natsConnectToken := &NATSConnectToken{
		Token: &ClientCredentialsTokenType{
			ClientID:     request.ClientID,
			ClientSecret: request.ClientSecret,
			Account:      request.Account,
		},
	}
	natsConnectTokenJson, _ := json.Marshal(natsConnectToken)
	return string(natsConnectTokenJson), nil
}
func CreateNatsConnectionWithClientCredentials(request *NATSConnectTokenClientCredentialsRequest) (*nats.Conn, error) {
	natsConnectTokenJson, _ := CreateNATSConnectTokenClientCredentials(request)
	tokenHandler := nats.TokenHandler(func() string {
		return string(natsConnectTokenJson)
	})
	nc, err := nats.Connect(request.NATSUrl, tokenHandler)
	return nc, err
}
func DecodeNATSConnectTokenClientCredentials(natsConnectTokenJson string) (*ClientCredentialsTokenType, error) {
	natsConnectToken := &NATSConnectToken{}
	err := json.Unmarshal([]byte(natsConnectTokenJson), natsConnectToken)
	if err != nil {
		return nil, err
	}
	rr, ok := natsConnectToken.Token.(*ClientCredentialsTokenType)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid token type")
	}
	return rr, nil
}
