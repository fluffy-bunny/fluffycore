package cefilter

import (
	"fmt"

	"github.com/fluffy-bunny/fluffycore/cloudevent/contracts"
)

// FromCloudEvent builds an Attrs map from a CloudEvent-like value.
// Pass your cloudevents.Event via the CloudEventAccessor adapter so this
// package stays free of the SDK import — swap the accessor for any event type.
//
// Usage:
//
//	attrs := cefilter.FromCloudEvent(cefilter.SDKEvent(event))
//	match := compiled.Match(attrs)
type CloudEventAccessor interface {
	// Standard context attributes
	Type() string
	Source() string
	Subject() string
	ID() string
	// Extensions returns all custom attributes as map[string]any
	Extensions() map[string]any
}

// FromCloudEvent converts a CloudEvent to the flat Attrs map.
// Custom extension values are coerced to strings via fmt.Sprintf.
func FromCloudEvent(e CloudEventAccessor) Attrs {
	a := Attrs{
		"type":   e.Type(),
		"source": e.Source(),
		"id":     e.ID(),
	}
	if s := e.Subject(); s != "" {
		a["subject"] = s
	}
	for k, v := range e.Extensions() {
		a[k] = fmt.Sprintf("%v", v)
	}
	return a
}

// FilterSet holds a collection of named, pre-compiled filters and matches
// an event against all of them in one pass.
type FilterSet struct {
	// MaxMatches caps how many entries Match() will collect.
	// When the limit is reached, Match() stops immediately and calls OnLimitExceeded.
	// 0 means unlimited (default).
	MaxMatches int

	// OnLimitExceeded is called when Match() stops early due to MaxMatches.
	// attrs is the event that triggered it; count is the number of matches collected
	// before stopping. Typical use: point this at your slog/zap logger.
	// Called at most once per Match() invocation.
	OnLimitExceeded func(attrs Attrs, count int)

	filters []namedFilter
}

type namedFilter struct {
	entry contracts.FilterEntry
	pred  Predicate
}

// Add compiles and registers a filter expression under the given entry.
// Returns an error if the expression is invalid — fail at startup, not per-event.
func (fs *FilterSet) Add(entry contracts.FilterEntry, expr string) error {
	pred, err := Parse(expr)
	if err != nil {
		return fmt.Errorf("filter %q: %w", entry.Name, err)
	}
	fs.filters = append(fs.filters, namedFilter{entry: entry, pred: pred})
	return nil
}

// Match returns the FilterEntry of every filter that matches the given attributes.
// Each entry carries the Name and Hint the caller registered — use Hint to route
// the event to its destination (NATS subject, handler, queue, etc.).
// If MaxMatches > 0 and the result set reaches that limit, Match stops collecting
// and calls OnLimitExceeded before returning the capped slice.
func (fs *FilterSet) Match(a Attrs) []contracts.FilterEntry {
	var matched []contracts.FilterEntry
	for _, f := range fs.filters {
		if f.pred.Match(a) {
			matched = append(matched, f.entry)
			if fs.MaxMatches > 0 && len(matched) >= fs.MaxMatches {
				if fs.OnLimitExceeded != nil {
					fs.OnLimitExceeded(a, len(matched))
				}
				break
			}
		}
	}
	return matched
}

// MatchAny returns true as soon as the first filter matches (short-circuits).
func (fs *FilterSet) MatchAny(a Attrs) bool {
	for _, f := range fs.filters {
		if f.pred.Match(a) {
			return true
		}
	}
	return false
}

// MatchEndpoints returns the distinct, non-empty Hint values of all matching filters.
// If multiple filter rules share the same Hint (endpoint URL, NATS subject, etc.),
// that endpoint appears only once — safe to iterate and dispatch without dedup logic.
// Respects MaxMatches (treated as max distinct endpoints); calls OnLimitExceeded when hit.
func (fs *FilterSet) MatchEndpoints(a Attrs) []string {
	seen := make(map[string]struct{})
	var endpoints []string
	for _, f := range fs.filters {
		if f.pred.Match(a) {
			hint := f.entry.Hint
			if hint == "" {
				continue
			}
			if _, dup := seen[hint]; dup {
				continue
			}
			seen[hint] = struct{}{}
			endpoints = append(endpoints, hint)
			if fs.MaxMatches > 0 && len(endpoints) >= fs.MaxMatches {
				if fs.OnLimitExceeded != nil {
					fs.OnLimitExceeded(a, len(endpoints))
				}
				break
			}
		}
	}
	return endpoints
}

// Remove unregisters the filter with the given Name. No-op if not found.
func (fs *FilterSet) Remove(name string) {
	kept := fs.filters[:0]
	for _, f := range fs.filters {
		if f.entry.Name != name {
			kept = append(kept, f)
		}
	}
	fs.filters = kept
}
