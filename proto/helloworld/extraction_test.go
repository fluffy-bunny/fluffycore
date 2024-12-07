package helloworld

import (
	"encoding/json"
	"testing"

	"github.com/mdaverde/jsonpath"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestExtraction(t *testing.T) {
	request := &ParentMessage{
		NestedMessage: &NestedMessage{
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

}
