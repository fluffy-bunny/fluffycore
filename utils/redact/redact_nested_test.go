package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type nestedSecret struct {
	APIKey string `redact:"true"`
}

type configWithNested struct {
	Name   string
	Nested *nestedSecret
	List   []*nestedSecret
	Values []nestedSecret
}

func newNestedConfig() *configWithNested {
	return &configWithNested{
		Name:   "svc",
		Nested: &nestedSecret{APIKey: "nested-api-key-123"},
		List:   []*nestedSecret{{APIKey: "list-api-key-456"}},
		Values: []nestedSecret{{APIKey: "value-api-key-789"}},
	}
}

// The copy was shallow, so redacting it rewrote the source's nested secrets.
func TestMakeRedactedCopy_DoesNotModifySource(t *testing.T) {
	src := newNestedConfig()
	dst := &configWithNested{}
	require.NoError(t, MakeRedactedCopy(src, dst))

	require.Equal(t, "nested-api-key-123", src.Nested.APIKey)
	require.Equal(t, "list-api-key-456", src.List[0].APIKey)
	require.Equal(t, "value-api-key-789", src.Values[0].APIKey)

	require.NotEqual(t, src.Nested.APIKey, dst.Nested.APIKey)
	require.NotEqual(t, src.List[0].APIKey, dst.List[0].APIKey)
	require.NotEqual(t, src.Values[0].APIKey, dst.Values[0].APIKey)
}

func TestCloneAndRedact_DoesNotModifySource(t *testing.T) {
	src := newNestedConfig()
	_, err := CloneAndRedact(src)
	require.NoError(t, err)
	require.Equal(t, "nested-api-key-123", src.Nested.APIKey)
	require.Equal(t, "list-api-key-456", src.List[0].APIKey)
}
