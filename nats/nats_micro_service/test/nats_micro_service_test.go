package test

import (
	"encoding/json"
	"testing"

	nats_micro_service "github.com/fluffy-bunny/fluffycore/nats/nats_micro_service"
	"github.com/fluffy-bunny/fluffycore/proto/helloworld"
	"github.com/mdaverde/jsonpath"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestExtractRouteParams(t *testing.T) {
	route := "org.1234"
	parameterizedRoute := "org.${orgId}"
	params, err := nats_micro_service.ExtractRouteParams(route, parameterizedRoute)
	require.NoError(t, err)
	require.Equal(t, 1, len(params))
	require.Equal(t, "1234", params["orgId"])

	route = "org.1234.user.5678"
	parameterizedRoute = "org.${orgId}.user.${userId}"
	params, err = nats_micro_service.ExtractRouteParams(route, parameterizedRoute)
	require.NoError(t, err)
	require.Equal(t, 2, len(params))
	require.Equal(t, "1234", params["orgId"])

	route = "org.1234.user.5678"
	parameterizedRoute = "org.${orgId}.user.${userId}.b.${b}"
	params, err = nats_micro_service.ExtractRouteParams(route, parameterizedRoute)
	require.Error(t, err)
	require.Nil(t, params)

	route = "org.1234.user.5678"
	parameterizedRoute = "org"
	params, err = nats_micro_service.ExtractRouteParams(route, parameterizedRoute)
	require.Error(t, err)
	require.Nil(t, params)

	route = "org.1234.user.5678"
	parameterizedRoute = "org.${orgId}.user.5678"
	params, err = nats_micro_service.ExtractRouteParams(route, parameterizedRoute)
	require.NoError(t, err)
	require.Equal(t, 1, len(params))
	require.Equal(t, "1234", params["orgId"])

	route = "org.1234.user.5678"
	parameterizedRoute = "org.${orgId}.user.5678.b.c"
	params, err = nats_micro_service.ExtractRouteParams(route, parameterizedRoute)
	require.Error(t, err)
	require.Nil(t, params)

}
func TestInject(t *testing.T) {
	request := &helloworld.ParentMessage{
		NestedMessage: &helloworld.NestedMessage{
			OrgId: "1234567890",
			Age:   1,
		},
		OrgId: "1234567890",
		Age:   1,
	}
	rr, err := nats_micro_service.InjectParameterizedRoutesIntoProtoMessage(
		"norg.org1.nage.23.org.org1.age.23",
		"norg.${orgId}.nage.${age}.org.${nestedMessage.orgId}.age.${nestedMessage.age}",
		request)
	require.NoError(t, err)
	require.NotNil(t, rr)

	// type cast
	rr2, ok := rr.(*helloworld.ParentMessage)
	require.True(t, ok)
	require.NotNil(t, rr2)
	require.Equal(t, "org1", rr2.OrgId)
	require.Equal(t, int32(23), rr2.Age)
	require.Equal(t, "org1", rr2.NestedMessage.OrgId)
	require.Equal(t, int32(23), rr2.NestedMessage.Age)
}

func TestExtraction(t *testing.T) {
	request := &helloworld.ParentMessage{
		NestedMessage: &helloworld.NestedMessage{
			OrgId: "1234567890",
			Age:   1,
		},
	}
	pJson, err := protojson.Marshal(request)
	require.NoError(t, err)

	var payload interface{}

	err = json.Unmarshal([]byte(pJson), &payload)
	require.NoError(t, err)

	value, err := jsonpath.Get(payload, "nestedMessage.orgId")
	require.NoError(t, err)
	require.Equal(t, "1234567890", value)

	value, err = jsonpath.Get(payload, "nestedMessage.age")
	require.NoError(t, err)
	require.Equal(t, float64(1), value)

	rr := &helloworld.HelloRequest{
		Name:  "world",
		OrgId: "1234567890",
	}
	final, err := nats_micro_service.ReplaceTokens("org.${nestedMessage.orgId}", request)
	require.NoError(t, err)
	require.Equal(t, "org.1234567890", final)

	final, err = nats_micro_service.ReplaceTokens("org.${orgId}", rr)
	require.NoError(t, err)
	require.Equal(t, "org.1234567890", final)

}
