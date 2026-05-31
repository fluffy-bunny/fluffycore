package cefilter

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/fluffy-bunny/fluffycore/cloudevent/contracts"
)

// IndexedFilterSet is safe for concurrent use. Filters can be added at any
// time without pausing the event stream.
//
// Read path  (MatchAny / Match): RLock — multiple goroutines read in parallel.
// Write path (Add):              Lock  — O(1) map insert, held for microseconds.
//
// Add is O(1); the previous copy-on-write design was O(N) per add because it
// cloned the entire exactIndex map, which hangs for 100K entries.
//
// With 100K type-constrained filters the hot path for an unwanted event is:
//
// RLock + 1 hash miss + RUnlock ≈ 25 ns
type IndexedFilterSet struct {
	// MaxMatches caps how many entries Match() will collect.
	// When the limit is reached, Match() stops immediately and calls OnLimitExceeded.
	// 0 means unlimited (default).
	// Set before concurrent use begins — not protected by the internal RWMutex.
	MaxMatches int

	// OnLimitExceeded is called when Match() stops early due to MaxMatches.
	// attrs is the event that triggered it; count is the number of matches collected.
	// Called at most once per Match() invocation.
	// Set before concurrent use begins — not protected by the internal RWMutex.
	OnLimitExceeded func(attrs Attrs, count int)

	mu              sync.RWMutex
	exactIndex      map[string][]*indexedEntry
	globEntries     []*globIndexEntry
	wildcardEntries []*indexedEntry
}

type indexedEntry struct {
	entry contracts.FilterEntry
	pred  Predicate
}

type globIndexEntry struct {
	rx    *regexp.Regexp
	entry *indexedEntry
}

// Add compiles and dynamically registers a filter expression under the given entry.
// O(1) — takes an exclusive lock only long enough to do a map insert.
// Safe to call concurrently with MatchAny / Match.
func (fs *IndexedFilterSet) Add(entry contracts.FilterEntry, expr string) error {
	pred, err := Parse(expr)
	if err != nil {
		return fmt.Errorf("filter %q: %w", entry.Name, err)
	}
	ie := &indexedEntry{entry: entry, pred: pred}
	tc := extractTypeConstraint(pred)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.exactIndex == nil {
		fs.exactIndex = make(map[string][]*indexedEntry)
	}
	switch {
	case tc.unconstrained:
		fs.wildcardEntries = append(fs.wildcardEntries, ie)
	default:
		for v := range tc.exact {
			fs.exactIndex[v] = append(fs.exactIndex[v], ie)
		}
		for _, rx := range tc.globs {
			fs.globEntries = append(fs.globEntries, &globIndexEntry{rx: rx, entry: ie})
		}
		if len(tc.exact) == 0 && len(tc.globs) == 0 {
			fs.wildcardEntries = append(fs.wildcardEntries, ie)
		}
	}
	return nil
}

// MatchAny returns true as soon as the first filter matches.
// Holds a shared read-lock; multiple goroutines can call this in parallel.
// For an event type that no filter cares about: RLock + 1 hash miss + RUnlock.
func (fs *IndexedFilterSet) MatchAny(a Attrs) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	t := a["type"]
	for _, e := range fs.exactIndex[t] {
		if e.pred.Match(a) {
			return true
		}
	}
	for _, ge := range fs.globEntries {
		if ge.rx.MatchString(t) && ge.entry.pred.Match(a) {
			return true
		}
	}
	for _, e := range fs.wildcardEntries {
		if e.pred.Match(a) {
			return true
		}
	}
	return false
}

// Match returns the FilterEntry of every filter that matched.
// Each entry carries the Name and Hint the caller registered — use Hint
// for fan-out: publish the event to each returned hint destination.
// If MaxMatches > 0 and the result set reaches that limit, Match stops and
// calls OnLimitExceeded before returning the capped slice.
func (fs *IndexedFilterSet) Match(a Attrs) []contracts.FilterEntry {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	t := a["type"]
	seen := make(map[string]struct{})
	var matched []contracts.FilterEntry
	done := false

	add := func(e *indexedEntry) {
		if done {
			return
		}
		if _, dup := seen[e.entry.Name]; dup {
			return
		}
		seen[e.entry.Name] = struct{}{}
		if e.pred.Match(a) {
			matched = append(matched, e.entry)
			if fs.MaxMatches > 0 && len(matched) >= fs.MaxMatches {
				if fs.OnLimitExceeded != nil {
					fs.OnLimitExceeded(a, len(matched))
				}
				done = true
			}
		}
	}
	for _, e := range fs.exactIndex[t] {
		if done {
			break
		}
		add(e)
	}
	for _, ge := range fs.globEntries {
		if done {
			break
		}
		if ge.rx.MatchString(t) {
			add(ge.entry)
		}
	}
	for _, e := range fs.wildcardEntries {
		if done {
			break
		}
		add(e)
	}
	return matched
}

// MatchEndpoints returns the distinct, non-empty Hint values of all matching filters.
// If multiple filter rules share the same Hint (endpoint URL, NATS subject, etc.),
// that endpoint appears only once — iterate and dispatch without extra dedup logic.
// Respects MaxMatches (treated as max distinct endpoints); calls OnLimitExceeded when hit.
func (fs *IndexedFilterSet) MatchEndpoints(a Attrs) []string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	t := a["type"]
	seenNames := make(map[string]struct{})
	seenHints := make(map[string]struct{})
	var endpoints []string
	done := false

	add := func(e *indexedEntry) {
		if done {
			return
		}
		if _, dup := seenNames[e.entry.Name]; dup {
			return
		}
		seenNames[e.entry.Name] = struct{}{}
		hint := e.entry.Hint
		if hint == "" {
			return
		}
		if _, dup := seenHints[hint]; dup {
			return // already dispatching to this endpoint
		}
		if e.pred.Match(a) {
			seenHints[hint] = struct{}{}
			endpoints = append(endpoints, hint)
			if fs.MaxMatches > 0 && len(endpoints) >= fs.MaxMatches {
				if fs.OnLimitExceeded != nil {
					fs.OnLimitExceeded(a, len(endpoints))
				}
				done = true
			}
		}
	}
	for _, e := range fs.exactIndex[t] {
		if done {
			break
		}
		add(e)
	}
	for _, ge := range fs.globEntries {
		if done {
			break
		}
		if ge.rx.MatchString(t) {
			add(ge.entry)
		}
	}
	for _, e := range fs.wildcardEntries {
		if done {
			break
		}
		add(e)
	}
	return endpoints
}

// Remove unregisters the filter with the given Name. No-op if not found.
// Safe to call concurrently with MatchAny / Match / MatchEndpoints.
func (fs *IndexedFilterSet) Remove(name string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for k, entries := range fs.exactIndex {
		kept := entries[:0]
		for _, e := range entries {
			if e.entry.Name != name {
				kept = append(kept, e)
			}
		}
		if len(kept) == 0 {
			delete(fs.exactIndex, k)
		} else {
			fs.exactIndex[k] = kept
		}
	}

	keptGlob := fs.globEntries[:0]
	for _, ge := range fs.globEntries {
		if ge.entry.entry.Name != name {
			keptGlob = append(keptGlob, ge)
		}
	}
	fs.globEntries = keptGlob

	keptWild := fs.wildcardEntries[:0]
	for _, e := range fs.wildcardEntries {
		if e.entry.Name != name {
			keptWild = append(keptWild, e)
		}
	}
	fs.wildcardEntries = keptWild
}

// Stats returns a summary useful for capacity planning.
func (fs *IndexedFilterSet) Stats() IndexStats {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	seen := make(map[*indexedEntry]struct{})
	for _, entries := range fs.exactIndex {
		for _, e := range entries {
			seen[e] = struct{}{}
		}
	}
	for _, ge := range fs.globEntries {
		seen[ge.entry] = struct{}{}
	}
	return IndexStats{
		Total:            len(seen) + len(fs.wildcardEntries),
		ExactTypeBuckets: len(fs.exactIndex),
		GlobEntries:      len(fs.globEntries),
		Unconstrained:    len(fs.wildcardEntries),
	}
}

// IndexStats describes how filters are distributed across index buckets.
type IndexStats struct {
	Total            int // total registered filters
	ExactTypeBuckets int // number of distinct exact type values indexed
	GlobEntries      int // filters with a LIKE type constraint
	Unconstrained    int // filters with no type constraint (O(N) cost each event)
}

// -------------------------------------------------------------------------
// Type-constraint extractor — walks the predicate tree once at Add time.
// -------------------------------------------------------------------------

type typeConstraint struct {
	exact         map[string]struct{}
	globs         []*regexp.Regexp
	unconstrained bool // any type could satisfy this predicate
}

func extractTypeConstraint(pred Predicate) typeConstraint {
	switch p := pred.(type) {

	case *eqPred:
		if p.key == "type" {
			return typeConstraint{exact: map[string]struct{}{p.val: {}}}
		}
		return typeConstraint{unconstrained: true}

	case *nePred:
		// != inverts: any type except p.val could match — treat as unconstrained.
		return typeConstraint{unconstrained: true}

	case *likePred:
		if p.key == "type" {
			return typeConstraint{globs: []*regexp.Regexp{p.rx}}
		}
		return typeConstraint{unconstrained: true}

	case *inPred:
		if p.key == "type" {
			exact := make(map[string]struct{}, len(p.vals))
			for v := range p.vals {
				exact[v] = struct{}{}
			}
			return typeConstraint{exact: exact}
		}
		return typeConstraint{unconstrained: true}

	case *existsPred:
		return typeConstraint{unconstrained: true}

	case *andPred:
		// AND: type must satisfy BOTH sides — take the intersection.
		// If one side is unconstrained, the other side's constraint governs.
		lc := extractTypeConstraint(p.left)
		rc := extractTypeConstraint(p.right)
		if lc.unconstrained {
			return rc
		}
		if rc.unconstrained {
			return lc
		}
		return intersectConstraints(lc, rc)

	case *orPred:
		// OR: type must satisfy EITHER side — take the union.
		// If either side is unconstrained, the OR can fire on any type.
		lc := extractTypeConstraint(p.left)
		rc := extractTypeConstraint(p.right)
		if lc.unconstrained || rc.unconstrained {
			return typeConstraint{unconstrained: true}
		}
		return unionConstraints(lc, rc)

	case *notPred:
		// NOT inverts the constraint; conservative: treat as unconstrained.
		return typeConstraint{unconstrained: true}

	default:
		return typeConstraint{unconstrained: true}
	}
}

func intersectConstraints(a, b typeConstraint) typeConstraint {
	result := typeConstraint{exact: make(map[string]struct{})}
	// exact ∩ exact
	for v := range a.exact {
		if _, ok := b.exact[v]; ok {
			result.exact[v] = struct{}{}
		}
	}
	// exact values from a that match a glob in b
	for v := range a.exact {
		for _, rx := range b.globs {
			if rx.MatchString(v) {
				result.exact[v] = struct{}{}
			}
		}
	}
	// exact values from b that match a glob in a
	for v := range b.exact {
		for _, rx := range a.globs {
			if rx.MatchString(v) {
				result.exact[v] = struct{}{}
			}
		}
	}
	// globs ∩ globs: include both as a superset (safe: false positives ok)
	result.globs = append(a.globs, b.globs...)
	return result
}

func unionConstraints(a, b typeConstraint) typeConstraint {
	result := typeConstraint{exact: make(map[string]struct{})}
	for v := range a.exact {
		result.exact[v] = struct{}{}
	}
	for v := range b.exact {
		result.exact[v] = struct{}{}
	}
	result.globs = append(a.globs, b.globs...)
	return result
}
