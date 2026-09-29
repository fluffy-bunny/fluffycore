package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_propertybag "github.com/fluffy-bunny/fluffycore/contracts/propertybag"
	fluffycore_services_propertybag "github.com/fluffy-bunny/fluffycore/services/propertybag"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func newBag(t *testing.T) fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag {
	t.Helper()
	b := di.Builder()
	fluffycore_services_propertybag.AddScopedRequestContextLoggingPropertyBag(b)
	scope := di.Get[di.ScopeFactory](b.Build()).CreateScope()
	t.Cleanup(scope.Dispose)
	return di.Get[fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag](scope.Container())
}

// The hook no longer goes through json.Marshal; the "ctx" object must look the same.
func TestPropertyBagHook_OutputShape(t *testing.T) {
	bag := newBag(t)
	bag.Set("sub", "user-1")
	bag.Set("aud", []string{"a", "b"})
	bag.Set("origin", map[string]interface{}{"app": "x", "n": 1.0})

	var buf bytes.Buffer
	l := zerolog.New(&buf).Hook(PropertyBagHook{PropertyBag: bag})
	l.Info().Msg("m")
	var line map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))
	require.Equal(t, map[string]interface{}{
		"sub": "user-1", "aud": []interface{}{"a", "b"}, "origin": map[string]interface{}{"app": "x", "n": 1.0},
	}, line["ctx"])

	buf.Reset()
	l = zerolog.New(&buf).Hook(PropertyBagHook{PropertyBag: newBag(t)})
	l.Info().Msg("m")
	require.NotContains(t, buf.String(), `"ctx"`, "an empty bag adds nothing")
}
