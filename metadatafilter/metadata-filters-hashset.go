package metadatafilter

import (
	"github.com/fluffy-bunny/fluffycore/gods/sets/hashset"
	wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
)

// NewHeaderSet ...
func NewHeaderSet() *hashset.StringSet {
	set := hashset.NewStringSet(wellknown.MetaDataFilter...)
	return set
}
