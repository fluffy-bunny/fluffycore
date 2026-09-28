package cefilter

import (
	"testing"

	"github.com/fluffy-bunny/fluffycore/cloudevent/contracts"
)

// An updated rule used to be appended next to its previous version. Match dedups by Name
// and keeps the FIRST entry it sees, so the stale version kept matching (and the new one
// could be shadowed entirely).
func TestIndexedFilterSet_AddSameNameReplaces(t *testing.T) {
	var fs IndexedFilterSet
	mustAdd := func(hint, expr string) {
		t.Helper()
		if err := fs.Add(contracts.FilterEntry{Name: "rule", Hint: hint}, expr); err != nil {
			t.Fatal(err)
		}
	}
	mustAdd("v1", `type = "old.type"`)
	mustAdd("v2", `type = "new.type"`)

	if fs.MatchAny(Attrs{"type": "old.type"}) {
		t.Fatal("the previous version of the rule still matches")
	}
	got := fs.Match(Attrs{"type": "new.type"})
	if len(got) != 1 || got[0].Hint != "v2" {
		t.Fatalf("want the new version only, got %+v", got)
	}
	if n := fs.Stats().Total; n != 1 {
		t.Fatalf("want 1 registered filter, got %d", n)
	}

	// Same type bucket, changed predicate and hint.
	mustAdd("v3", `type = "new.type" AND region = "eu"`)
	if fs.MatchAny(Attrs{"type": "new.type", "region": "us"}) {
		t.Fatal("v2's predicate still matches")
	}
	if got := fs.MatchEndpoints(Attrs{"type": "new.type", "region": "eu"}); len(got) != 1 || got[0] != "v3" {
		t.Fatalf("want [v3], got %v", got)
	}

	// Re-adding after Remove works, and a failed parse leaves the current version in place.
	fs.Remove("rule")
	if fs.Stats().Total != 0 {
		t.Fatal("Remove left entries behind")
	}
	mustAdd("v4", `type = "x"`)
	if err := fs.Add(contracts.FilterEntry{Name: "rule", Hint: "bad"}, `type = `); err == nil {
		t.Fatal("expected a parse error")
	}
	if got := fs.MatchEndpoints(Attrs{"type": "x"}); len(got) != 1 || got[0] != "v4" {
		t.Fatalf("a failed Add must not remove the current version, got %v", got)
	}
}
