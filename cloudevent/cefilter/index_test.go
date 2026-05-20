package cefilter_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/fluffy-bunny/fluffycore/cloudevent/cefilter"
	"github.com/fluffy-bunny/fluffycore/cloudevent/contracts"
)

func TestIndexedFilterSet_QuickReject(t *testing.T) {
	var fs cefilter.IndexedFilterSet
	fs.Add(contracts.FilterEntry{Name: "order-org1"}, `type = "com.example.order" AND orgid = "org-1"`)
	fs.Add(contracts.FilterEntry{Name: "order-org2"}, `type = "com.example.order" AND orgid = "org-2"`)
	fs.Add(contracts.FilterEntry{Name: "payment-any"}, `type = "com.example.payment"`)

	// Event nobody cares about — type not in any filter
	unknown := cefilter.Attrs{"type": "com.nobody.cares", "orgid": "org-1"}
	if fs.MatchAny(unknown) {
		t.Error("expected quick-reject for unknown type")
	}

	// Known type, wrong org
	wrongOrg := cefilter.Attrs{"type": "com.example.order", "orgid": "org-99"}
	if fs.MatchAny(wrongOrg) {
		t.Error("expected no match: known type, wrong org")
	}

	// Matching event
	match := cefilter.Attrs{"type": "com.example.order", "orgid": "org-1"}
	if !fs.MatchAny(match) {
		t.Error("expected match for order + org-1")
	}
}

func TestIndexedFilterSet_Match_ReturnsAllMatches(t *testing.T) {
	var fs cefilter.IndexedFilterSet
	fs.Add(contracts.FilterEntry{Name: "hook-org1"}, `type = "com.example.order" AND orgid = "org-1"`)
	fs.Add(contracts.FilterEntry{Name: "hook-org2"}, `type = "com.example.order" AND orgid = "org-2"`)
	fs.Add(contracts.FilterEntry{Name: "hook-all-orders"}, `type = "com.example.order"`)

	attrs := cefilter.Attrs{"type": "com.example.order", "orgid": "org-1"}
	got := fs.Match(attrs)
	if len(got) != 2 { // hook-org1 + hook-all-orders
		t.Errorf("expected 2 matches, got %d: %v", len(got), got)
	}
}

func TestIndexedFilterSet_GlobType(t *testing.T) {
	var fs cefilter.IndexedFilterSet
	fs.Add(contracts.FilterEntry{Name: "glob-filter"}, `type LIKE "com.example.*" AND orgid = "org-1"`)

	match := cefilter.Attrs{"type": "com.example.order", "orgid": "org-1"}
	if !fs.MatchAny(match) {
		t.Error("expected glob match")
	}

	noMatch := cefilter.Attrs{"type": "com.other.order", "orgid": "org-1"}
	if fs.MatchAny(noMatch) {
		t.Error("expected no match: type doesn't match glob")
	}
}

func TestIndexedFilterSet_InType(t *testing.T) {
	var fs cefilter.IndexedFilterSet
	fs.Add(contracts.FilterEntry{Name: "multi-type"}, `type IN ("com.example.order", "com.example.payment") AND orgid = "org-1"`)

	for _, typ := range []string{"com.example.order", "com.example.payment"} {
		attrs := cefilter.Attrs{"type": typ, "orgid": "org-1"}
		if !fs.MatchAny(attrs) {
			t.Errorf("expected match for type %q", typ)
		}
	}
	unknown := cefilter.Attrs{"type": "com.example.refund", "orgid": "org-1"}
	if fs.MatchAny(unknown) {
		t.Error("expected quick-reject for type not in IN list")
	}
}

func TestIndexedFilterSet_OrType(t *testing.T) {
	var fs cefilter.IndexedFilterSet
	fs.Add(contracts.FilterEntry{Name: "or-filter"}, `(type = "com.example.order" OR type = "com.example.payment") AND orgid = "org-1"`)

	if !fs.MatchAny(cefilter.Attrs{"type": "com.example.order", "orgid": "org-1"}) {
		t.Error("expected match for order")
	}
	if !fs.MatchAny(cefilter.Attrs{"type": "com.example.payment", "orgid": "org-1"}) {
		t.Error("expected match for payment")
	}
	if fs.MatchAny(cefilter.Attrs{"type": "com.example.refund", "orgid": "org-1"}) {
		t.Error("expected quick-reject for refund")
	}
}

func TestIndexedFilterSet_Unconstrained(t *testing.T) {
	var fs cefilter.IndexedFilterSet
	fs.Add(contracts.FilterEntry{Name: "any-with-priority"}, `EXISTS priority`) // no type constraint

	stats := fs.Stats()
	if stats.Unconstrained != 1 {
		t.Errorf("expected 1 unconstrained filter, got %d", stats.Unconstrained)
	}
	// Still works
	if !fs.MatchAny(cefilter.Attrs{"type": "anything", "priority": "high"}) {
		t.Error("expected match")
	}
}

func TestIndexedFilterSet_Stats(t *testing.T) {
	var fs cefilter.IndexedFilterSet
	fs.Add(contracts.FilterEntry{Name: "f1"}, `type = "com.example.order"`)
	fs.Add(contracts.FilterEntry{Name: "f2"}, `type = "com.example.payment"`)
	fs.Add(contracts.FilterEntry{Name: "f3"}, `type LIKE "com.acme.*"`)
	fs.Add(contracts.FilterEntry{Name: "f4"}, `EXISTS priority`)

	stats := fs.Stats()
	fmt.Printf("IndexedFilterSet stats: %+v\n", stats)
	if stats.ExactTypeBuckets != 2 {
		t.Errorf("expected 2 exact buckets, got %d", stats.ExactTypeBuckets)
	}
	if stats.GlobEntries != 1 {
		t.Errorf("expected 1 glob entry, got %d", stats.GlobEntries)
	}
	if stats.Unconstrained != 1 {
		t.Errorf("expected 1 unconstrained, got %d", stats.Unconstrained)
	}
}

func TestIndexedFilterSet_ConcurrentAddAndMatch(t *testing.T) {
	var fs cefilter.IndexedFilterSet

	// Seed some initial filters.
	for i := range 10 {
		fs.Add(contracts.FilterEntry{Name: fmt.Sprintf("seed-%d", i)}, fmt.Sprintf(`type = "com.example.type%d"`, i))
	}

	var wg sync.WaitGroup

	// Goroutines continuously adding new filters (dynamic registration).
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := range 20 {
				name := fmt.Sprintf("dynamic-%d-%d", i, j)
				expr := fmt.Sprintf(`type = "com.dynamic.%d.%d"`, i, j)
				if err := fs.Add(contracts.FilterEntry{Name: name}, expr); err != nil {
					t.Errorf("Add(%q): %v", name, err)
				}
			}
		}(i)
	}

	// Goroutines continuously matching events — must not race with Add.
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event := cefilter.Attrs{"type": "com.example.type3"}
			for range 100 {
				fs.MatchAny(event)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkFilterSet_MatchAny_NoMatch(b *testing.B) {
	var fs cefilter.FilterSet
	for i := range 1000 {
		fs.Add(contracts.FilterEntry{Name: fmt.Sprintf("f%d", i)}, fmt.Sprintf(`type = "com.example.type%d" AND orgid = "org-%d"`, i, i))
	}
	event := cefilter.Attrs{"type": "com.nobody.cares", "orgid": "org-0"}
	b.ResetTimer()
	for range b.N {
		fs.MatchAny(event)
	}
}

func BenchmarkIndexedFilterSet_MatchAny_NoMatch(b *testing.B) {
	var fs cefilter.IndexedFilterSet
	for i := range 1000 {
		fs.Add(contracts.FilterEntry{Name: fmt.Sprintf("f%d", i)}, fmt.Sprintf(`type = "com.example.type%d" AND orgid = "org-%d"`, i, i))
	}
	event := cefilter.Attrs{"type": "com.nobody.cares", "orgid": "org-0"}
	b.ResetTimer()
	for range b.N {
		fs.MatchAny(event)
	}
}
