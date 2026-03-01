package metadatafilter

import (
	"testing"

	wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder_Empty(t *testing.T) {
	b := NewEntryPointAllowedMetadataMapBuilder()
	require.NotNil(t, b)
	require.Empty(t, b.EntryPointAllowedMetadataMap)
}

func TestWithAllowedMetadataHeader_Single(t *testing.T) {
	b := NewEntryPointAllowedMetadataMapBuilder().
		WithAllowedMetadataHeader("/my.Service/Method", "x-custom-header")

	set, ok := b.EntryPointAllowedMetadataMap["/my.Service/Method"]
	require.True(t, ok)
	require.True(t, set.Contains("x-custom-header"))
}

func TestWithAllowedMetadataHeader_MultipleCalls(t *testing.T) {
	b := NewEntryPointAllowedMetadataMapBuilder().
		WithAllowedMetadataHeader("/svc/A", "header-1").
		WithAllowedMetadataHeader("/svc/A", "header-2").
		WithAllowedMetadataHeader("/svc/B", "header-3")

	setA, ok := b.EntryPointAllowedMetadataMap["/svc/A"]
	require.True(t, ok)
	require.True(t, setA.Contains("header-1"))
	require.True(t, setA.Contains("header-2"))
	require.Equal(t, 2, setA.Size())

	setB, ok := b.EntryPointAllowedMetadataMap["/svc/B"]
	require.True(t, ok)
	require.True(t, setB.Contains("header-3"))
	require.Equal(t, 1, setB.Size())
}

func TestWithAllowedMetadataHeader_Lowercases(t *testing.T) {
	b := NewEntryPointAllowedMetadataMapBuilder().
		WithAllowedMetadataHeader("/svc/M", "X-My-Header", "Content-Type")

	set := b.EntryPointAllowedMetadataMap["/svc/M"]
	require.True(t, set.Contains("x-my-header"))
	require.True(t, set.Contains("content-type"))
	require.False(t, set.Contains("X-My-Header"))
}

func TestWithAllowedMetadataHeader_MultipleHeaders(t *testing.T) {
	b := NewEntryPointAllowedMetadataMapBuilder().
		WithAllowedMetadataHeader("/svc/M", "a", "b", "c")

	set := b.EntryPointAllowedMetadataMap["/svc/M"]
	require.Equal(t, 3, set.Size())
	require.True(t, set.Contains("a"))
	require.True(t, set.Contains("b"))
	require.True(t, set.Contains("c"))
}

func TestNewHeaderSet_ContainsWellknown(t *testing.T) {
	set := NewHeaderSet()
	require.NotNil(t, set)
	require.False(t, set.Empty())

	// Verify it contains the entries from wellknown.MetaDataFilter
	for _, h := range wellknown.MetaDataFilter {
		require.True(t, set.Contains(h), "expected header %q in set", h)
	}
}

func TestNewHeaderSet_SizeMatchesFilter(t *testing.T) {
	set := NewHeaderSet()
	require.Equal(t, len(wellknown.MetaDataFilter), set.Size())
}

func TestNewHeaderSet_ContainsAuthorization(t *testing.T) {
	set := NewHeaderSet()
	require.True(t, set.Contains("authorization"))
	require.True(t, set.Contains("content-type"))
}
